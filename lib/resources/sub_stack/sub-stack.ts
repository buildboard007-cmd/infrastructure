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
        const userManagementIntegration = new LambdaIntegration(this.lambdaConstruct.userManagementLambda);
        const issueManagementIntegration = new LambdaIntegration(this.lambdaConstruct.issueManagementLambda);
        const rfiManagementIntegration = new LambdaIntegration(this.lambdaConstruct.rfiManagementLambda);
        const corsIntegration = new LambdaIntegration(this.lambdaConstruct.corsLambda);

        // Create /org resource with Cognito authorization
        const orgResource = this.api.root.addResource('org');
        orgResource.addMethod('GET', orgManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        orgResource.addMethod('PUT', orgManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        orgResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

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

        // Create /users resource with Cognito authorization
        const usersResource = this.api.root.addResource('users');
        usersResource.addMethod('GET', userManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        usersResource.addMethod('POST', userManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        usersResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /users/{userId} resource for specific user operations
        const userIdResource = usersResource.addResource('{userId}');
        userIdResource.addMethod('GET', userManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        userIdResource.addMethod('PUT', userManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        userIdResource.addMethod('DELETE', userManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        userIdResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /users/{userId}/reset-password resource for password reset
        const userPasswordResetResource = userIdResource.addResource('reset-password');
        userPasswordResetResource.addMethod('PATCH', userManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        userPasswordResetResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Add issue management routes
        // Create /projects/{projectId}/issues resource for issue management
        const projectIssuesResource = projectIdResource.addResource('issues');
        projectIssuesResource.addMethod('GET', issueManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectIssuesResource.addMethod('POST', issueManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectIssuesResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /issues resource for direct issue operations
        const issuesResource = this.api.root.addResource('issues');
        issuesResource.addMethod('POST', issueManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        issuesResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /issues/{issueId} resource for specific issue operations
        const issueIdResource = issuesResource.addResource('{issueId}');
        issueIdResource.addMethod('GET', issueManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        issueIdResource.addMethod('PUT', issueManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        issueIdResource.addMethod('DELETE', issueManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        issueIdResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /issues/{issueId}/status resource for status-only updates
        const issueStatusResource = issueIdResource.addResource('status');
        issueStatusResource.addMethod('PATCH', issueManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        issueStatusResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Add RFI management routes
        // Create /projects/{projectId}/rfis resource for RFI management
        const projectRFIsResource = projectIdResource.addResource('rfis');
        projectRFIsResource.addMethod('GET', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectRFIsResource.addMethod('POST', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectRFIsResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /rfis resource for direct RFI operations
        const rfisResource = this.api.root.addResource('rfis');
        rfisResource.addMethod('POST', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        rfisResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /rfis/{rfiId} resource for specific RFI operations
        const rfiIdResource = rfisResource.addResource('{rfiId}');
        rfiIdResource.addMethod('GET', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        rfiIdResource.addMethod('PUT', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        rfiIdResource.addMethod('DELETE', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        rfiIdResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /rfis/{rfiId}/status resource for status-only updates
        const rfiStatusResource = rfiIdResource.addResource('status');
        rfiStatusResource.addMethod('PATCH', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        rfiStatusResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /rfis/{rfiId}/submit resource for submitting RFI
        const rfiSubmitResource = rfiIdResource.addResource('submit');
        rfiSubmitResource.addMethod('PATCH', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        rfiSubmitResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /rfis/{rfiId}/respond resource for responding to RFI
        const rfiRespondResource = rfiIdResource.addResource('respond');
        rfiRespondResource.addMethod('PATCH', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        rfiRespondResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /rfis/{rfiId}/approve resource for approving RFI
        const rfiApproveResource = rfiIdResource.addResource('approve');
        rfiApproveResource.addMethod('PATCH', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        rfiApproveResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /rfis/{rfiId}/reject resource for rejecting RFI
        const rfiRejectResource = rfiIdResource.addResource('reject');
        rfiRejectResource.addMethod('PATCH', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        rfiRejectResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /rfis/{rfiId}/attachments resource for RFI attachment management
        const rfiAttachmentsResource = rfiIdResource.addResource('attachments');
        rfiAttachmentsResource.addMethod('GET', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        rfiAttachmentsResource.addMethod('POST', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        rfiAttachmentsResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /rfis/{rfiId}/attachments/{attachmentId} resource for specific attachment operations
        const rfiAttachmentIdResource = rfiAttachmentsResource.addResource('{attachmentId}');
        rfiAttachmentIdResource.addMethod('GET', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        rfiAttachmentIdResource.addMethod('DELETE', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        rfiAttachmentIdResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

        // Create /rfis/{rfiId}/comments resource for RFI comment management
        const rfiCommentsResource = rfiIdResource.addResource('comments');
        rfiCommentsResource.addMethod('GET', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        rfiCommentsResource.addMethod('POST', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        rfiCommentsResource.addMethod('OPTIONS', corsIntegration); // OPTIONS doesn't need auth for CORS

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
