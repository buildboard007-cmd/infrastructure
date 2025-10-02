import * as cdk from "aws-cdk-lib";
import {NestedStack, NestedStackProps} from "aws-cdk-lib";
import {findStageOption as findStageOptions, StackOptions} from "../../types/stack-options";
import {StageEnvironment} from "../../types/stage-environment";
import {Construct} from "constructs";
import {KeyConstruct} from "../key_construct/key-construct";
import {LambdaConstruct} from "../lambda_construct/lambda-construct";
import {LambdaConstructProps} from "../../types/lambda-construct-props";
import {CognitoConstruct} from "../cognito_construct/cognito-construct";
import {BasePathMapping, DomainName, RestApi, LambdaIntegration, CognitoUserPoolsAuthorizer, Cors} from "aws-cdk-lib/aws-apigateway";
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
        
        // Create API Gateway with CORS enabled at gateway level
        this.api = new RestApi(this, "InfrastructureAPI", {
            restApiName: props.options.apiName,
            description: props.options.apiName,
            deployOptions: {
                stageName: props.options.apiStageName,
            },
            defaultCorsPreflightOptions: {
                allowOrigins: Cors.ALL_ORIGINS,
                allowMethods: Cors.ALL_METHODS,
                allowHeaders: [
                    'Content-Type',
                    'X-Amz-Date',
                    'Authorization',
                    'X-Api-Key',
                    'X-Amz-Security-Token',
                    'X-Amz-User-Agent'
                ]
            }
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
        const submittalManagementIntegration = new LambdaIntegration(this.lambdaConstruct.submittalManagementLambda);
        const assignmentManagementIntegration = new LambdaIntegration(this.lambdaConstruct.assignmentManagementLambda);
        // CORS Lambda integration removed - using API Gateway CORS instead

        // Create /org resource with Cognito authorization
        const orgResource = this.api.root.addResource('org');
        orgResource.addMethod('GET', orgManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        orgResource.addMethod('PUT', orgManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // Create /locations resource with Cognito authorization
        const locationsResource = this.api.root.addResource('locations');
        locationsResource.addMethod('GET', locationManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        locationsResource.addMethod('POST', locationManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // Create /locations/{id} resource for specific location operations
        const locationIdResource = locationsResource.addResource('{id}');
        locationIdResource.addMethod('GET', locationManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        locationIdResource.addMethod('PUT', locationManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // locationIdResource.addMethod('DELETE', locationManagementIntegration, {
        //     authorizer: cognitoAuthorizer
        // }); // Temporarily commented to avoid API Gateway limits
        // CORS handled at API Gateway level

        // Create /roles resource with Cognito authorization
        const rolesResource = this.api.root.addResource('roles');
        rolesResource.addMethod('GET', rolesManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        rolesResource.addMethod('POST', rolesManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // Create /roles/{id} resource for specific role operations
        const roleIdResource = rolesResource.addResource('{id}');
        roleIdResource.addMethod('GET', rolesManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        roleIdResource.addMethod('PUT', rolesManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // roleIdResource.addMethod('DELETE', rolesManagementIntegration, {
        //     authorizer: cognitoAuthorizer
        // }); // Temporarily commented to avoid API Gateway limits
        // CORS handled at API Gateway level

        // Create /roles/{id}/permissions resource for role-permission management
        const rolePermissionsResource = roleIdResource.addResource('permissions');
        rolePermissionsResource.addMethod('POST', rolesManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // rolePermissionsResource.addMethod('DELETE', rolesManagementIntegration, {
        //     authorizer: cognitoAuthorizer
        // }); // Temporarily commented to avoid API Gateway limits
        // CORS handled at API Gateway level

        // Create /permissions resource with Cognito authorization
        const permissionsResource = this.api.root.addResource('permissions');
        permissionsResource.addMethod('GET', permissionsManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        permissionsResource.addMethod('POST', permissionsManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // Create /permissions/{id} resource for specific permission operations
        const permissionIdResource = permissionsResource.addResource('{id}');
        permissionIdResource.addMethod('GET', permissionsManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        permissionIdResource.addMethod('PUT', permissionsManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // permissionIdResource.addMethod('DELETE', permissionsManagementIntegration, {
        //     authorizer: cognitoAuthorizer
        // }); // Temporarily commented to avoid API Gateway limits
        // CORS handled at API Gateway level

        // Create /projects resource with Cognito authorization
        const projectsResource = this.api.root.addResource('projects');
        projectsResource.addMethod('GET', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectsResource.addMethod('POST', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // Create /projects/{projectId} resource for specific project operations
        const projectIdResource = projectsResource.addResource('{projectId}');
        projectIdResource.addMethod('GET', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectIdResource.addMethod('PUT', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // projectIdResource.addMethod('DELETE', projectManagementIntegration, {
        //     authorizer: cognitoAuthorizer
        // }); // Temporarily commented to avoid API Gateway limits
        // CORS handled at API Gateway level


        // Create /projects/{projectId}/attachments resource for project attachment management
        const projectAttachmentsResource = projectIdResource.addResource('attachments');
        projectAttachmentsResource.addMethod('GET', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectAttachmentsResource.addMethod('POST', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // Create /projects/{projectId}/attachments/{attachmentId} resource for specific attachment operations
        const projectAttachmentIdResource = projectAttachmentsResource.addResource('{attachmentId}');
        projectAttachmentIdResource.addMethod('GET', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // projectAttachmentIdResource.addMethod('DELETE', projectManagementIntegration, {
        //     authorizer: cognitoAuthorizer
        // }); // Temporarily commented to avoid API Gateway limits
        // CORS handled at API Gateway level

        // Create /projects/{projectId}/users resource for project user role management
        const projectUsersResource = projectIdResource.addResource('users');
        projectUsersResource.addMethod('GET', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectUsersResource.addMethod('POST', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // Create /projects/{projectId}/users/{assignmentId} resource for specific user role operations
        const projectUserAssignmentIdResource = projectUsersResource.addResource('{assignmentId}');
        projectUserAssignmentIdResource.addMethod('PUT', projectManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // projectUserAssignmentIdResource.addMethod('DELETE', projectManagementIntegration, {
        //     authorizer: cognitoAuthorizer
        // }); // Temporarily commented to avoid API Gateway limits
        // CORS handled at API Gateway level

        // Create /users resource with Cognito authorization
        const usersResource = this.api.root.addResource('users');
        usersResource.addMethod('GET', userManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        usersResource.addMethod('POST', userManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // Create /users/{userId} resource for specific user operations
        const userIdResource = usersResource.addResource('{userId}');
        userIdResource.addMethod('GET', userManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        userIdResource.addMethod('PUT', userManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // userIdResource.addMethod('DELETE', userManagementIntegration, {
        //     authorizer: cognitoAuthorizer
        // }); // Temporarily commented to avoid API Gateway limits
        // CORS handled at API Gateway level

        // Create /users/{userId}/reset-password resource for password reset
        const userPasswordResetResource = userIdResource.addResource('reset-password');
        userPasswordResetResource.addMethod('PATCH', userManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // Create /users/{userId}/location resource for location selection updates
        const userLocationResource = userIdResource.addResource('location');
        userLocationResource.addMethod('PATCH', userManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // Create /users/{userId}/selected-location/{locationId} resource for setting user's selected location preference
        const userSelectedLocationResource = userIdResource.addResource('selected-location');
        const userSelectedLocationIdResource = userSelectedLocationResource.addResource('{locationId}');
        userSelectedLocationIdResource.addMethod('PUT', userManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // Add issue management routes
        // Create /projects/{projectId}/issues resource for issue management
        const projectIssuesResource = projectIdResource.addResource('issues');
        projectIssuesResource.addMethod('GET', issueManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        projectIssuesResource.addMethod('POST', issueManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // Create /issues resource for direct issue operations
        const issuesResource = this.api.root.addResource('issues');
        issuesResource.addMethod('POST', issueManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // Create /issues/{issueId} resource for specific issue operations
        const issueIdResource = issuesResource.addResource('{issueId}');
        issueIdResource.addMethod('GET', issueManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        issueIdResource.addMethod('PUT', issueManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // issueIdResource.addMethod('DELETE', issueManagementIntegration, {
        //     authorizer: cognitoAuthorizer
        // }); // Temporarily commented to avoid API Gateway limits
        // CORS handled at API Gateway level

        // Create /issues/{issueId}/status resource for status-only updates
        const issueStatusResource = issueIdResource.addResource('status');
        issueStatusResource.addMethod('PATCH', issueManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // CONSOLIDATED RFI MANAGEMENT (6 endpoints total)

        // Core RFI CRUD operations
        const rfisResource = this.api.root.addResource('rfis');
        rfisResource.addMethod('POST', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        const rfiIdResource = rfisResource.addResource('{rfiId}');
        rfiIdResource.addMethod('GET', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        rfiIdResource.addMethod('PUT', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // Sub-resource operations
        const rfiAttachmentsResource = rfiIdResource.addResource('attachments');
        rfiAttachmentsResource.addMethod('POST', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });

        const rfiCommentsResource = rfiIdResource.addResource('comments');
        rfiCommentsResource.addMethod('POST', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // Create /assignments resource for direct assignment operations
        const assignmentsResource = this.api.root.addResource('assignments');
        assignmentsResource.addMethod('POST', assignmentManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // Create /assignments/{assignmentId} resource for specific assignment operations
        const assignmentIdResource = assignmentsResource.addResource('{assignmentId}');
        assignmentIdResource.addMethod('GET', assignmentManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        assignmentIdResource.addMethod('PUT', assignmentManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        assignmentIdResource.addMethod('DELETE', assignmentManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // Shared /contexts resource for both RFI and assignment queries
        const contextsResource = this.api.root.addResource('contexts');
        const contextTypeResource = contextsResource.addResource('{contextType}');
        const contextIdResource = contextTypeResource.addResource('{contextId}');

        // Context-based RFI queries (replaces /projects/{projectId}/rfis)
        const contextRfisResource = contextIdResource.addResource('rfis');
        contextRfisResource.addMethod('GET', rfiManagementIntegration, {
            authorizer: cognitoAuthorizer
        });

        // Context-based assignment queries (project team queries)
        const contextAssignmentsResource = contextIdResource.addResource('assignments');
        contextAssignmentsResource.addMethod('GET', assignmentManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // Context-based submittal queries (replaces /projects/{projectId}/submittals)
        const contextSubmittalsResource = contextIdResource.addResource('submittals');
        contextSubmittalsResource.addMethod('GET', submittalManagementIntegration, {
            authorizer: cognitoAuthorizer
        });

        // Context submittal stats
        const contextSubmittalStatsResource = contextSubmittalsResource.addResource('stats');
        contextSubmittalStatsResource.addMethod('GET', submittalManagementIntegration, {
            authorizer: cognitoAuthorizer
        });

        // Context submittal export
        const contextSubmittalExportResource = contextSubmittalsResource.addResource('export');
        contextSubmittalExportResource.addMethod('GET', submittalManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // CONSOLIDATED SUBMITTAL MANAGEMENT (10 endpoints total)

        // Core submittal CRUD operations
        const submittalsResource = this.api.root.addResource('submittals');
        submittalsResource.addMethod('POST', submittalManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        const submittalIdResource = submittalsResource.addResource('{submittalId}');
        submittalIdResource.addMethod('GET', submittalManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        submittalIdResource.addMethod('PUT', submittalManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // Submittal workflow operations
        const submittalWorkflowResource = submittalIdResource.addResource('workflow');
        submittalWorkflowResource.addMethod('POST', submittalManagementIntegration, {
            authorizer: cognitoAuthorizer
        });
        // CORS handled at API Gateway level

        // Submittal attachments
        const submittalAttachmentsResource = submittalIdResource.addResource('attachments');
        submittalAttachmentsResource.addMethod('POST', submittalManagementIntegration, {
            authorizer: cognitoAuthorizer
        });

        // CORS handled at API Gateway level

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
