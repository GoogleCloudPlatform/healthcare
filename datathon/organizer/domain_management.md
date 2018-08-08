# Domain Management

For Google Cloud Platform (GCP) projects with shared access among multiple users
(e.g. projects for access-controlled data sharing or for datathon hosting), it is
recommended to have a domain associated to the project. There are two benefits
in doing so.

1.  In order for a GCP project to use a group as its owner, the project must be
    associated to a domain. The reason that a group owner is preferred to an
    individual owner is that it is safer to have backup owners in case the
    primary owner is absent, and it is much easier to manage group membership
    than individual ownership.

1.  There is a set up group management API to programmatically manage membership
    for GSuite groups and groups created in domains with Cloud Identity. This is
    useful when a large number of users need to be managed at regular basis. In
    contrast, public Google groups only has a web UI for manual operations.

## Obtaining a Google Managed Domain

If you are a [GSuite](https://gsuite.google.com/) customer, you organization
should already have a domain that is managed by Google services. Contact your
organization administrator to grant you necessary access for the GCP project
setup.

If you are not a GSuite customer, but own a domain that you would like to enable
Google domain management, you can opt in to the free
[Google Cloud Identity](https://cloud.google.com/identity/) service. Simply
follow instructions at https://support.google.com/a/topic/7386475 to get
started.

## Group Management API

The [Directory API](https://developers.google.com/admin-sdk/directory/) in the
GSuite Admin SDK can be used for both GSuite and Cloud Identity managed domains.
In particular, there is a set of
[group management](https://developers.google.com/admin-sdk/directory/v1/guides/manage-groups)
and
[group membership management](https://developers.google.com/admin-sdk/directory/v1/guides/manage-group-members)
RESTful APIs that can be used to do things like creating a group and adding
members to a group programmatically.

As an example, here are the steps to authenticate using OAuth 2.0, and then
issuing the API requests using `curl` in a CLI. This can be wrapped in programs
using various popular programming languages that the GSuite Admin SDK supports.

### Setting up Group Management APIs for group creation
1.  Create an OAuth client ID for API authentication. Go to the
    [APIs & Services > Credentials](https://console.cloud.google.com/apis/credentials) of an
    existing GCP project, and select "Create Credentials" => "OAuth Client ID".
    In "application type" screen, choose the "Other" radio button, enter a name
    for the client ID, and then click "Create". Once the client ID is created,
    download the JSON key file by clicking the download button at the right end
    of the Client ID record. Please keep this file in a private location, and we
    shall use it to get an access token in the next step.

1.  Generate the access token from the client ID JSON file. For this step, you
    need the `oauth2l` open-source software, which can be obtained by following
    this [instruction](https://github.com/google/oauth2l/blob/master/README.md).
    Then, run the following command to obtain the access token:

    ```shell
    oauth2l fetch --json <client_id JSON file> \
        https://www.googleapis.com/auth/admin.directory.group \
        https://www.googleapis.com/auth/apps.groups.settings
    ```

    This command will generate a URL to be opened in a browser, where you can
    log in to your account in the domain to manage, and grant the requested
    access. Once that is done, it will display a string code, which you can copy
    and paste to the `oauth2l` command prompt. The tool should then print the
    access token string. Please store it in the `TOKEN` variable for later use.

    ```shell
    TOKEN=ya29...
    ```
#### Testing Group Management API Configuration
*   Now we are ready to use the log-in identity in the previous step to perform
    API calls. First let us list the current groups in your domain using a
    [`list` request](https://developers.google.com/admin-sdk/directory/v1/reference/groups/list).
    The `domain` parameter is required, which takes the value of your domain
    name. If you have not set the DOMAIN and PROJECT_PREFIX variables in this
    environment in an earlier step, please do so as well. For example,

    ```shell
    DOMAIN=datathon.com
    PROJECT_PREFIX=my-project
    ```

    ```shell
    curl \
      --header "Authorization: Bearer ${TOKEN}" \
      https://www.googleapis.com/admin/directory/v1/groups/?domain=$DOMAIN
    ```

    This should return an empty list unless groups were created before in the
    domain.

    ```
    {
     "kind": "admin#directory#groups",
     "etag": "\"some-eTag\""
    }
    ```

#### Advanced Testing
If you are following the Permission Control Group Setup from the README, you
can return to that document now. If you would like to test or explore the Group 
Management API further, proceed with these steps.

*   To create a new group, use the
    [`insert` method](https://developers.google.com/admin-sdk/directory/v1/reference/groups/insert)
    by issuing the following request

    ```shell
    curl -X POST  --header "Content-Type: application/json" \
        --header "Authorization: Bearer ${TOKEN}" \
        --data '{"email":"my-test@datathon.com"}' \
        https://www.googleapis.com/admin/directory/v1/groups
    ```

    If the request is successful, the request will return a JSON object with
    information about the newly created group.

    ```
    {
     "kind": "admin#directory#group",
     "id": "02u6wntf3l330xw",
     "etag": "\"fvhsVnsP-Q_A0ZHbc-PlLgJbgzQ/afsmxorRJLiR1xmtt1kU6hB-MUU\"",
     "email": "my-test@datathon.com",
     "name": "my-test",
     "description": "",
     "adminCreated": true
    }
    ```

    Note that the `name` and `description` fields are set to default values. You
    can specify them in your request JSON, in order to set them to more
    meaningful values upon creation. At this point, if you re-issue the `list`
    request above, you should be able to see this new group in the response.

    At the group-level, these methods are supported

    *   [`delete`](https://developers.google.com/admin-sdk/directory/v1/reference/groups/delete)
    *   [`get`](https://developers.google.com/admin-sdk/directory/v1/reference/groups/get)
    *   [`insert`](https://developers.google.com/admin-sdk/directory/v1/reference/groups/insert)
    *   [`list`](https://developers.google.com/admin-sdk/directory/v1/reference/groups/list)
    *   [`patch`](https://developers.google.com/admin-sdk/directory/v1/reference/groups/patch)
    *   [`update`](https://developers.google.com/admin-sdk/directory/v1/reference/groups/update)

*   Now that the group is created, let us try to add a new member to the group
    using the
    [`insert` method](https://developers.google.com/admin-sdk/directory/v1/reference/members/list)
    in the
    [group members API](https://developers.google.com/admin-sdk/directory/v1/reference/members).
    For this request, you need to specify in the request body JSON object the
    email address of the new member, as well as its `role` in the group (one of
    `MEMBER`, `MANAGER` or `OWNER`). The following request adds
    `member-email@gmail.com` into the group we created in the previous step
    (identified by the `groupKey` of `02u6wntf3l330xw` in the request URL) as a
    regular `member`.

    ```shell
    curl -X POST --header "Content-Type: application/json" \
        --header "Authorization: Bearer ${TOKEN}" \
        --data '{"role":"MEMBER","email":"member-email@gmail.com"}' \
        https://www.googleapis.com/admin/directory/v1/groups/02u6wntf3l330xw/members
    ```

    At the member-level, these methods are supported

    *   [`delete`](https://developers.google.com/admin-sdk/directory/v1/reference/members/delete)
    *   [`get`](https://developers.google.com/admin-sdk/directory/v1/reference/members/get)
    *   [`hasMember`](https://developers.google.com/admin-sdk/directory/v1/reference/members/delete)
    *   [`insert`](https://developers.google.com/admin-sdk/directory/v1/reference/members/insert)
    *   [`list`](https://developers.google.com/admin-sdk/directory/v1/reference/members/list)
    *   [`patch`](https://developers.google.com/admin-sdk/directory/v1/reference/members/patch)
    *   [`update`](https://developers.google.com/admin-sdk/directory/v1/reference/members/update)

With the group-level and member-level API methods, it is possible to achieve all
group management tasks using the RESTful API. To integrate the functionalities
in your program, you can also use one of the client libraries in any of the
supported programming languages (e.g. .NET, Go, Java, NodeJS, etc). You can find
more details about programming using the directory API following the
[Getting Started guide](https://developers.google.com/admin-sdk/directory/v1/get-start/getting-started).
