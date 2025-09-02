import {NestedStack, NestedStackProps} from "aws-cdk-lib";
import {findStageOption as findStageOptions, StackOptions} from "../../types/stack-options";
import {StageEnvironment} from "../../types/stage-environment";
import {Construct} from "constructs";
import {KeyConstruct} from "../key_construct/key-construct";
import {LambdaConstruct} from "../lambda_construct/lambda-construct";
import {LambdaConstructProps} from "../../types/lambda-construct-props";
import {CognitoConstruct} from "../cognito_construct/cognito-construct";
import {BasePathMapping, DomainName, RestApi, LambdaIntegration, CognitoUserPoolsAuthorizer} from "aws-cdk-lib/aws-apigateway";
import {GetAccountId} from "../../utils/account-utils";

interface KeyProps extends NestedStackProps {
    options: StackOptions;
    stageEnvironment: StageEnvironment;
}

export class SubStack extends NestedStack {
    private readonly keyConstruct: KeyConstruct;
    private readonly lambdaConstruct: LambdaConstruct;
    private readonly api: RestApi;

    constructor(scope: Construct, id: string, props: KeyProps) {
        super(scope, id, props);

        // this.keyConstruct = new KeyConstruct(this, 'KeyConstruct', {
        //     stageEnvironment: props.stageEnvironment,
        // });

        const lambdaConstructProps: LambdaConstructProps = {
            options: props.options,
            stageEnvironment: props.stageEnvironment
        };

        this.lambdaConstruct = new LambdaConstruct(this, "LambdaConstruct", lambdaConstructProps);

        const cognitoConstruct = new CognitoConstruct(this, "CognitoConstruct", {
            options: props.options,
            tokenCustomizerLambda: this.lambdaConstruct.tokenCustomizerLambda,
            userSignupLambda: this.lambdaConstruct.userSignupLambda,
            stage: props.stageEnvironment,
        });

        const account = GetAccountId(props.stageEnvironment);
        
        // Create API Gateway programmatically to avoid deployment dependency issues
        this.api = new RestApi(this, "InfrastructureAPI", {
            restApiName: props.options.apiName,
            description: props.options.apiName,
            deployOptions: {
                stageName: props.options.apiStageName,
            },
        });

        // Create Cognito User Pool authorizer
        const cognitoAuthorizer = new CognitoUserPoolsAuthorizer(this, 'CognitoAuthorizer', {
            cognitoUserPools: [cognitoConstruct.userPool],
            authorizerName: 'CognitoAuthorizer'
        });

        // Create Lambda integrations
        const orgManagementIntegration = new LambdaIntegration(this.lambdaConstruct.organizationManagementLambda);
        const locationManagementIntegration = new LambdaIntegration(this.lambdaConstruct.locationManagementLambda);
        const rolesManagementIntegration = new LambdaIntegration(this.lambdaConstruct.rolesManagementLambda);
        const permissionsManagementIntegration = new LambdaIntegration(this.lambdaConstruct.permissionsManagementLambda);
        const projectManagementIntegration = new LambdaIntegration(this.lambdaConstruct.projectManagementLambda);
        const corsIntegration = new LambdaIntegration(this.lambdaConstruct.corsLambda);

        // Create /org resource with Cognito authorization (legacy endpoint)
        const orgResource = this.api.root.addResource('org');
        orgResource.addMethod('GET', orgManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        orgResource.addMethod('PUT', orgManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        orgResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /organizations resource with Cognito authorization
        const organizationsResource = this.api.root.addResource('organizations');
        organizationsResource.addMethod('GET', orgManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        organizationsResource.addMethod('POST', orgManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        organizationsResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /organizations/{id} resource for specific organization operations
        const organizationIdResource = organizationsResource.addResource('{id}');
        organizationIdResource.addMethod('GET', orgManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        organizationIdResource.addMethod('PUT', orgManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        organizationIdResource.addMethod('DELETE', orgManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        organizationIdResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /locations resource with Cognito authorization
        const locationsResource = this.api.root.addResource('locations');
        locationsResource.addMethod('GET', locationManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        locationsResource.addMethod('POST', locationManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        locationsResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /locations/{id} resource for specific location operations
        const locationIdResource = locationsResource.addResource('{id}');
        locationIdResource.addMethod('GET', locationManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        locationIdResource.addMethod('PUT', locationManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        locationIdResource.addMethod('DELETE', locationManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        locationIdResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /roles resource with Cognito authorization
        const rolesResource = this.api.root.addResource('roles');
        rolesResource.addMethod('GET', rolesManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        rolesResource.addMethod('POST', rolesManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        rolesResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /roles/{id} resource for specific role operations
        const roleIdResource = rolesResource.addResource('{id}');
        roleIdResource.addMethod('GET', rolesManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        roleIdResource.addMethod('PUT', rolesManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        roleIdResource.addMethod('DELETE', rolesManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        roleIdResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /roles/{id}/permissions resource for role-permission management
        const rolePermissionsResource = roleIdResource.addResource('permissions');
        rolePermissionsResource.addMethod('POST', rolesManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        rolePermissionsResource.addMethod('DELETE', rolesManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        rolePermissionsResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /permissions resource with Cognito authorization
        const permissionsResource = this.api.root.addResource('permissions');
        permissionsResource.addMethod('GET', permissionsManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        permissionsResource.addMethod('POST', permissionsManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        permissionsResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /permissions/{id} resource for specific permission operations
        const permissionIdResource = permissionsResource.addResource('{id}');
        permissionIdResource.addMethod('GET', permissionsManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        permissionIdResource.addMethod('PUT', permissionsManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        permissionIdResource.addMethod('DELETE', permissionsManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        permissionIdResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /projects resource with Cognito authorization
        const projectsResource = this.api.root.addResource('projects');
        projectsResource.addMethod('GET', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectsResource.addMethod('POST', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectsResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /projects/{projectId} resource for specific project operations
        const projectIdResource = projectsResource.addResource('{projectId}');
        projectIdResource.addMethod('GET', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectIdResource.addMethod('PUT', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectIdResource.addMethod('DELETE', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectIdResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /projects/{projectId}/managers resource for project manager management
        const projectManagersResource = projectIdResource.addResource('managers');
        projectManagersResource.addMethod('GET', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectManagersResource.addMethod('POST', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectManagersResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /projects/{projectId}/managers/{managerId} resource for specific manager operations
        const projectManagerIdResource = projectManagersResource.addResource('{managerId}');
        projectManagerIdResource.addMethod('GET', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectManagerIdResource.addMethod('PUT', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectManagerIdResource.addMethod('DELETE', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectManagerIdResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /projects/{projectId}/attachments resource for project attachment management
        const projectAttachmentsResource = projectIdResource.addResource('attachments');
        projectAttachmentsResource.addMethod('GET', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectAttachmentsResource.addMethod('POST', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectAttachmentsResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /projects/{projectId}/attachments/{attachmentId} resource for specific attachment operations
        const projectAttachmentIdResource = projectAttachmentsResource.addResource('{attachmentId}');
        projectAttachmentIdResource.addMethod('GET', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectAttachmentIdResource.addMethod('DELETE', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectAttachmentIdResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /projects/{projectId}/users resource for project user role management
        const projectUsersResource = projectIdResource.addResource('users');
        projectUsersResource.addMethod('GET', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectUsersResource.addMethod('POST', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectUsersResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /projects/{projectId}/users/{assignmentId} resource for specific user role operations
        const projectUserAssignmentIdResource = projectUsersResource.addResource('{assignmentId}');
        projectUserAssignmentIdResource.addMethod('PUT', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectUserAssignmentIdResource.addMethod('DELETE', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectUserAssignmentIdResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Skip domain for LOCAL
        if (props.stageEnvironment != StageEnvironment.LOCAL) {
            const stageOptions = findStageOptions(
                props.options,
                props.stageEnvironment
            );

            // Only create domain mapping if domain configuration is provided
            if (!stageOptions || !stageOptions.domainName || stageOptions.domainName.trim() === "") {
                console.log("No domain configuration found for stage, skipping domain mapping");
                return;
            }

            let domainName = DomainName.fromDomainNameAttributes(
                this,
                "APIDomainName",
                {
                    domainName: stageOptions.domainName,
                    domainNameAliasTarget: stageOptions.domainNameAliasTarget,
                    domainNameAliasHostedZoneId: stageOptions.domainNameAliasHostedZoneId,
                }
            );

            const basePathMapping = new BasePathMapping(this, "ApiBasePathMapping", {
                domainName: domainName,
                restApi: this.api,
                basePath: "infra",
                stage: this.api.deploymentStage,
            });
        }

    }

    get corsLambdaArn(): string {
        return this.lambdaConstruct.corsLambdaArn;
    }

    // get dataKey(): IKey {
    //     return this.keyConstruct.dataKey;
    // }

    // get snsSqsKey(): IKey {
    //     return this.keyConstruct.snsSqsKey;
    // }
}
