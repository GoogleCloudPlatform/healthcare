# Fhir Immunizations Demo

The goal of the application is to demostrate how to interact with a FHIR data
store from a Javascript application. The application itself allows users
to track the immunizations they have received and which ones may be coming up
for renewal. The application also provides immunization suggestions for
countries that the user indicates they will be travelling to.

# Development

The application is developed using the Angular CLI. More information about
that project can be found on their [website](https://cli.angular.io/).

Install the dependencies with npm by running `npm install`. Next, follow the
Cloud Healthcare
[quickstart](https://cloud.google.com/healthcare/docs/quickstart) to set up
your project. You will also need to setup OAuth for your project, following the
instructions [here](https://cloud.google.com/docs/authentication/end-user). Finally,
create a new file at `src/environments/environment.dev.ts` with the
configuration for your new project, replacing all the `TODO_`s with their
appropriate values.

```javascript
export const environment = {
  production: false,
  clientId: TODO_OAUTH_CLIENT_ID,
  fhirEndpoint: {
    baseURL: "https://healthcare.googleapis.com/v1beta1/",
    project: TODO_PROJECT,
    location: TODO_REGION,
    dataset: TODO_DATASET,
    fhirStore: TODO_FHIR_STORE
  }
};
```

The application can now be run with `ng serve`.
