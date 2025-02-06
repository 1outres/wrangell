# wrangell

This software provides a framework for building event-driven automation on Kubernetes. It allows users to define custom resources representing Events, Actions, and Triggers to facilitate flexible workflows.
 * Events represent occurrences or states, with user-defined schemas describing their data.
 * Actions specify operations to be executed when triggered, with customizable parameters.
 * Triggers evaluate conditions based on Event data and determine which Actions to execute.

Users can define these resources declaratively and deploy them within their Kubernetes cluster. At runtime, the controller listens for Events, evaluates Trigger conditions, and executes the corresponding Actions sequentially if the conditions are met.

Example Use Case:
	1.	Event: A custom program deployed in the cluster emits an Event when a Pod’s status changes.
	2.	Action: Another custom program sends a Webhook notification to an external service.
	3.	Trigger: A condition is defined to send a notification only when a Pod enters the “CrashLoopBackOff” state.

## Installation

```
kubectl apply -f https://raw.githubusercontent.com/1outres/wrangell/refs/heads/v2/config/install.yaml
```

