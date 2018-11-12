# Inference Module

The inference module is a component which listens for PubSub notifications and
prepares features and initiates online prediction. It is developed based on
Cloud Functions, please run the following command to deploy it:

```bash
./deploy.sh --name <NAME> \
            --topic <TOPIC> \
            --env_vars MODEL=<MODEL>,VERSION=<VERSION>
```

Please check the docs directory up one level for the meaning of each argument,
and how to train and deploy the model.