package amqp

import (
	"bytes"
	"context"
	"fmt"
	wrangellv1alpha1 "github.com/1outres/wrangell/api/v1alpha1"
	"github.com/Azure/go-amqp"
	ceamqp "github.com/cloudevents/sdk-go/protocol/amqp/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/expr-lang/expr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"net/url"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
)

type Receiver interface {
	Init() error
	Start(ctx context.Context) error
	Close(ctx context.Context) error
}

type receiver struct {
	uri        string
	protocol   *ceamqp.Protocol
	amqpClient cloudevents.Client
	kubeClient client.Client
	httpClient *http.Client
}

func (r *receiver) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)

	return r.amqpClient.StartReceiver(ctx, func(e cloudevents.Event) {
		var eventSpec wrangellv1alpha1.Event
		err := r.kubeClient.Get(ctx, client.ObjectKey{Name: e.Type()}, &eventSpec)
		if err != nil {
			if errors.IsNotFound(err) {
				logger.Info("Ignoring event %s, not found spec for type", "id", e.ID(), "source", e.Source(), "type", e.Type())
			} else {
				logger.Error(err, "Failed to get event spec", "id", e.ID(), "source", e.Source(), "type", e.Type())
			}
			return
		}

		if e.DataContentType() != "" && e.DataContentType() != "application/json" {
			logger.Info("Ignoring event, unsupported content type", "id", e.ID(), "source", e.Source(), "type", e.Type(), "content-type", e.DataContentType())
			return
		}

		var namespace string

		if namespace, exists := e.Extensions()["namespace"]; exists {
			if val, ok := namespace.(string); ok {
				namespace = val
			} else {
				logger.Info("Ignoring event, invalid namespace", "id", e.ID(), "source", e.Source(), "type", e.Type())
				return
			}
		} else {
			logger.Info("Ignoring event, missing namespace", "id", e.ID(), "source", e.Source(), "type", e.Type())
			return
		}

		// create data
		var rawData map[string]interface{}
		if err := e.DataAs(&rawData); err != nil {
			logger.Error(err, "Failed to unmarshal event data: "+err.Error(), "id", e.ID(), "source", e.Source(), "type", e.Type())
			return
		}
		data, err := eventSpec.Spec.Data.CreateDataFrom(rawData)
		if err != nil {
			logger.Error(err, "Failed to create data from event data: "+err.Error(), "id", e.ID(), "source", e.Source(), "type", e.Type())
			return
		}

		// run threads for each trigger
		var triggerList wrangellv1alpha1.TriggerList
		if err := r.kubeClient.List(ctx, &triggerList, &client.ListOptions{
			Namespace: namespace,
		}); err != nil {
			logger.Error(err, "Failed to list triggers", "id", e.ID(), "source", e.Source(), "type", e.Type())
			return
		}

		for _, trigger := range triggerList.Items {
			if trigger.Spec.Event != e.Type() {
				continue
			}

			go func() {
				history := wrangellv1alpha1.TriggerHistory{
					Time:   metav1.NewTime(e.Time()),
					Source: e.Source(),
					ID:     e.ID(),
					Message: []wrangellv1alpha1.TriggerHistoryMessage{
						{
							Time:    metav1.Now(),
							Success: true,
							Message: "Received an event",
						},
					},
				}

				defer func() {
					trigger.Status.History = append(trigger.Status.History, history)
					if err := r.kubeClient.Status().Update(ctx, &trigger); err != nil {
						logger.Error(err, "Failed to update trigger status", "id", e.ID(), "source", e.Source(), "type", e.Type())
					}
				}()

				// validate conditions
				for _, condition := range trigger.Spec.Conditions {
					program, err := expr.Compile(condition, expr.Env(data))
					if err != nil {
						history.AddMessage(false, "Failed to compile condition: "+err.Error())
						return
					}
					output, err := expr.Run(program, data)
					if err != nil {
						history.AddMessage(false, "Failed to run condition: "+err.Error())
						return
					}

					if output != true {
						history.AddMessage(false, "Condition is not satisfied")
						return
					}
				}
				history.AddMessage(true, "Conditions are satisfied")

				// run actions
				for _, action := range trigger.Spec.Actions {
					var actionSpec wrangellv1alpha1.Action
					if err := r.kubeClient.Get(ctx, client.ObjectKey{Name: action.Action}, &actionSpec); err != nil {
						history.AddMessage(false, "Internal Server Error")
						logger.Error(err, "Failed to get action spec", "id", e.ID(), "source", e.Source(), "type", e.Type(), "action", action.Action)
						return
					}

					// create params to post
					params := actionSpec.Spec.Data.CreateEmptyData()
					for _, param := range action.Params {
						var dataToStore interface{}
						if param.FromYaml != nil {
							err := json.Unmarshal(param.FromYaml.Raw, &dataToStore)
							if err != nil {
								history.AddMessage(false, "Internal Server Error")
								logger.Error(err, "Failed to unmarshal param from yaml", "id", e.ID(), "source", e.Source(), "type", e.Type(), "action", action.Action, "param", param.Name)
								return
							}
						} else if param.FromData != nil {
							value, exists := data.Get(param.FromData.Name)
							if !exists {
								history.AddMessage(false, "Failed to find param from event data: "+param.FromData.Name)
								logger.Info("Failed to find param from event data", "id", e.ID(), "source", e.Source(), "type", e.Type(), "action", action.Action, "param", param.Name)
								return
							}
							dataToStore = value
						}
						params.Set(param.Name, dataToStore)
					}

					rawParams, err := json.Marshal(params)
					if err != nil {
						history.AddMessage(false, "Internal Server Error")
						logger.Error(err, "Failed to marshal params", "id", e.ID(), "source", e.Source(), "type", e.Type(), "action", action.Action)
						return
					}

					// post a request
					req, err := http.NewRequest("POST", actionSpec.Spec.Endpoint, bytes.NewBuffer(rawParams))
					if err != nil {
						history.AddMessage(false, "Internal Server Error")
						logger.Error(err, "Failed to create request", "id", e.ID(), "source", e.Source(), "type", e.Type(), "action", action.Action)
						return
					}
					req.Header.Set("Content-Type", "application/json")

					resp, err := r.httpClient.Do(req)
					if err != nil {
						history.AddMessage(false, "Internal Server Error")
						logger.Error(err, "Failed to call an action", "id", e.ID(), "source", e.Source(), "type", e.Type(), "action", action.Action)
						return
					}

					if resp.StatusCode != 200 {
						history.AddMessage(false, "Action "+action.Action+" failed with status "+resp.Status)
						logger.Info("Failed to call an action", "id", e.ID(), "source", e.Source(), "type", e.Type(), "action", action.Action, "status", resp.StatusCode)
						return
					} else {
						history.AddMessage(true, "Action "+action.Action+" succeeded")
					}
				}
			}()
		}
	})
}

func (r *receiver) Close(ctx context.Context) error {
	return r.protocol.Close(ctx)
}

func (r *receiver) Init() error {
	host, node, opts := amqpConfig(r.uri)
	p, err := ceamqp.NewProtocol(host, node, []amqp.ConnOption{}, []amqp.SessionOption{}, opts...)
	if err != nil {
		return fmt.Errorf("failed to create AMQP protocol: %v", err)
	}
	r.protocol = p

	c, err := cloudevents.NewClient(p)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}
	r.amqpClient = c

	return nil
}

func NewReceiver(uri string, client client.Client) Receiver {
	return &receiver{
		uri:        uri,
		kubeClient: client,
		httpClient: &http.Client{},
	}
}

func amqpConfig(uri string) (server, node string, opts []ceamqp.Option) {
	u, _ := url.Parse(uri)
	if u.User != nil {
		user := u.User.Username()
		pass, _ := u.User.Password()
		opts = append(opts, ceamqp.WithConnOpt(amqp.ConnSASLPlain(user, pass)))
	}
	return uri, strings.TrimPrefix(u.Path, "/"), opts
}
