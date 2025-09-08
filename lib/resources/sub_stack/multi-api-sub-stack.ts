import {NestedStack, NestedStackProps} from "aws-cdk-lib";
import {findStageOption as findStageOptions, StackOptions} from "../../types/stack-options";
import {StageEnvironment} from "../../types/stage-environment";
import {Construct} from "constructs";
import {LambdaConstruct} from "../lambda_construct/lambda-construct";
import {LambdaConstructProps} from "../../types/lambda-construct-props";
import {CognitoConstruct} from "../cognito_construct/cognito-construct";
import {BasePathMapping, DomainName, RestApi, LambdaIntegration, CognitoUserPoolsAuthorizer} from "aws-cdk-lib/aws-apigateway";
import {GetAccountId} from "../../utils/account-utils";

interface MultiApiSubStackProps extends NestedStackProps {
    options: StackOptions;
    stageEnvironment: StageEnvironment;
}

export class MultiApiSubStack extends NestedStack {
    private readonly lambdaConstruct: LambdaConstruct;
    private readonly iamApi: RestApi;
    private readonly projectsApi: RestApi;
    private readonly issuesApi: RestApi;
    private readonly rfisApi: RestApi;

    constructor(scope: Construct, id: string, props: MultiApiSubStackProps) {
        super(scope, id, props);

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

        // Create Lambda integrations
        const corsIntegration = new LambdaIntegration(this.lambdaConstruct.corsLambda);

        // === IAM API ===
        this.iamApi = this.createIamApi(props, cognitoConstruct, corsIntegration);
        
        // === PROJECTS API ===
        this.projectsApi = this.createProjectsApi(props, cognitoConstruct, corsIntegration);
        
        // === ISSUES API ===
        this.issuesApi = this.createIssuesApi(props, cognitoConstruct, corsIntegration);
        
        // === RFIS API ===
        this.rfisApi = this.createRfisApi(props, cognitoConstruct, corsIntegration);

        // Setup domain mappings
        this.setupDomainMappings(props);
    }

    private createIamApi(props: MultiApiSubStackProps, cognitoConstruct: CognitoConstruct, corsIntegration: LambdaIntegration): RestApi {
        const iamApi = new RestApi(this, "IAMAPI", {
            restApiName: `${props.options.apiName}-IAM`,
            description: "IAM Management API - Users, Organization, Locations, Roles & Permissions",
            deployOptions: {
                stageName: props.options.apiStageName,
            },
        });

        // Create Cognito authorizer for this API
        const cognitoAuthorizer = new CognitoUserPoolsAuthorizer(this, 'IamApiCognitoAuthorizer', {
            cognitoUserPools: [cognitoConstruct.userPool],
            authorizerName: 'IamApiCognitoAuthorizer'
        });

        const orgManagementIntegration = new LambdaIntegration(this.lambdaConstruct.organizationManagementLambda);
        const userManagementIntegration = new LambdaIntegration(this.lambdaConstruct.userManagementLambda);
        const locationManagementIntegration = new LambdaIntegration(this.lambdaConstruct.locationManagementLambda);
        const rolesManagementIntegration = new LambdaIntegration(this.lambdaConstruct.rolesManagementLambda);
        const permissionsManagementIntegration = new LambdaIntegration(this.lambdaConstruct.permissionsManagementLambda);

        // Organization endpoints
        const orgResource = iamApi.root.addResource('org');
        orgResource.addMethod('GET', orgManagementIntegration, { authorizer: cognitoAuthorizer });
        orgResource.addMethod('PUT', orgManagementIntegration, { authorizer: cognitoAuthorizer });
        orgResource.addMethod('OPTIONS', corsIntegration);

        // User endpoints
        const usersResource = iamApi.root.addResource('users');
        usersResource.addMethod('GET', userManagementIntegration, { authorizer: cognitoAuthorizer });
        usersResource.addMethod('POST', userManagementIntegration, { authorizer: cognitoAuthorizer });
        usersResource.addMethod('OPTIONS', corsIntegration);

        const userIdResource = usersResource.addResource('{userId}');
        userIdResource.addMethod('GET', userManagementIntegration, { authorizer: cognitoAuthorizer });
        userIdResource.addMethod('PUT', userManagementIntegration, { authorizer: cognitoAuthorizer });
        userIdResource.addMethod('DELETE', userManagementIntegration, { authorizer: cognitoAuthorizer });
        userIdResource.addMethod('OPTIONS', corsIntegration);

        const userPasswordResetResource = userIdResource.addResource('reset-password');
        userPasswordResetResource.addMethod('PATCH', userManagementIntegration, { authorizer: cognitoAuthorizer });
        userPasswordResetResource.addMethod('OPTIONS', corsIntegration);

        // Location endpoints
        const locationsResource = iamApi.root.addResource('locations');
        locationsResource.addMethod('GET', locationManagementIntegration, { authorizer: cognitoAuthorizer });
        locationsResource.addMethod('POST', locationManagementIntegration, { authorizer: cognitoAuthorizer });
        locationsResource.addMethod('OPTIONS', corsIntegration);

        const locationIdResource = locationsResource.addResource('{id}');
        locationIdResource.addMethod('GET', locationManagementIntegration, { authorizer: cognitoAuthorizer });
        locationIdResource.addMethod('PUT', locationManagementIntegration, { authorizer: cognitoAuthorizer });
        locationIdResource.addMethod('DELETE', locationManagementIntegration, { authorizer: cognitoAuthorizer });
        locationIdResource.addMethod('OPTIONS', corsIntegration);

        // Roles endpoints
        const rolesResource = iamApi.root.addResource('roles');
        rolesResource.addMethod('GET', rolesManagementIntegration, { authorizer: cognitoAuthorizer });
        rolesResource.addMethod('POST', rolesManagementIntegration, { authorizer: cognitoAuthorizer });
        rolesResource.addMethod('OPTIONS', corsIntegration);

        const roleIdResource = rolesResource.addResource('{id}');
        roleIdResource.addMethod('GET', rolesManagementIntegration, { authorizer: cognitoAuthorizer });
        roleIdResource.addMethod('PUT', rolesManagementIntegration, { authorizer: cognitoAuthorizer });
        roleIdResource.addMethod('DELETE', rolesManagementIntegration, { authorizer: cognitoAuthorizer });
        roleIdResource.addMethod('OPTIONS', corsIntegration);

        const rolePermissionsResource = roleIdResource.addResource('permissions');
        rolePermissionsResource.addMethod('POST', rolesManagementIntegration, { authorizer: cognitoAuthorizer });
        rolePermissionsResource.addMethod('DELETE', rolesManagementIntegration, { authorizer: cognitoAuthorizer });
        rolePermissionsResource.addMethod('OPTIONS', corsIntegration);

        // Permissions endpoints
        const permissionsResource = iamApi.root.addResource('permissions');
        permissionsResource.addMethod('GET', permissionsManagementIntegration, { authorizer: cognitoAuthorizer });
        permissionsResource.addMethod('POST', permissionsManagementIntegration, { authorizer: cognitoAuthorizer });
        permissionsResource.addMethod('OPTIONS', corsIntegration);

        const permissionIdResource = permissionsResource.addResource('{id}');
        permissionIdResource.addMethod('GET', permissionsManagementIntegration, { authorizer: cognitoAuthorizer });
        permissionIdResource.addMethod('PUT', permissionsManagementIntegration, { authorizer: cognitoAuthorizer });
        permissionIdResource.addMethod('DELETE', permissionsManagementIntegration, { authorizer: cognitoAuthorizer });
        permissionIdResource.addMethod('OPTIONS', corsIntegration);

        return iamApi;
    }

    private createProjectsApi(props: MultiApiSubStackProps, cognitoConstruct: CognitoConstruct, corsIntegration: LambdaIntegration): RestApi {
        const projectsApi = new RestApi(this, "ProjectsAPI", {
            restApiName: `${props.options.apiName}-Projects`,
            description: "Project Management API",
            deployOptions: {
                stageName: props.options.apiStageName,
            },
        });

        // Create Cognito authorizer for this API
        const cognitoAuthorizer = new CognitoUserPoolsAuthorizer(this, 'ProjectsApiCognitoAuthorizer', {
            cognitoUserPools: [cognitoConstruct.userPool],
            authorizerName: 'ProjectsApiCognitoAuthorizer'
        });

        const projectManagementIntegration = new LambdaIntegration(this.lambdaConstruct.projectManagementLambda);

        // Projects endpoints
        const projectsResource = projectsApi.root.addResource('projects');
        projectsResource.addMethod('GET', projectManagementIntegration, { authorizer: cognitoAuthorizer });
        projectsResource.addMethod('POST', projectManagementIntegration, { authorizer: cognitoAuthorizer });
        projectsResource.addMethod('OPTIONS', corsIntegration);

        const projectIdResource = projectsResource.addResource('{projectId}');
        projectIdResource.addMethod('GET', projectManagementIntegration, { authorizer: cognitoAuthorizer });
        projectIdResource.addMethod('PUT', projectManagementIntegration, { authorizer: cognitoAuthorizer });
        projectIdResource.addMethod('DELETE', projectManagementIntegration, { authorizer: cognitoAuthorizer });
        projectIdResource.addMethod('OPTIONS', corsIntegration);

        // Project managers
        const projectManagersResource = projectIdResource.addResource('managers');
        projectManagersResource.addMethod('GET', projectManagementIntegration, { authorizer: cognitoAuthorizer });
        projectManagersResource.addMethod('POST', projectManagementIntegration, { authorizer: cognitoAuthorizer });
        projectManagersResource.addMethod('OPTIONS', corsIntegration);

        const projectManagerIdResource = projectManagersResource.addResource('{managerId}');
        projectManagerIdResource.addMethod('GET', projectManagementIntegration, { authorizer: cognitoAuthorizer });
        projectManagerIdResource.addMethod('PUT', projectManagementIntegration, { authorizer: cognitoAuthorizer });
        projectManagerIdResource.addMethod('DELETE', projectManagementIntegration, { authorizer: cognitoAuthorizer });
        projectManagerIdResource.addMethod('OPTIONS', corsIntegration);

        // Project attachments
        const projectAttachmentsResource = projectIdResource.addResource('attachments');
        projectAttachmentsResource.addMethod('GET', projectManagementIntegration, { authorizer: cognitoAuthorizer });
        projectAttachmentsResource.addMethod('POST', projectManagementIntegration, { authorizer: cognitoAuthorizer });
        projectAttachmentsResource.addMethod('OPTIONS', corsIntegration);

        const projectAttachmentIdResource = projectAttachmentsResource.addResource('{attachmentId}');
        projectAttachmentIdResource.addMethod('GET', projectManagementIntegration, { authorizer: cognitoAuthorizer });
        projectAttachmentIdResource.addMethod('DELETE', projectManagementIntegration, { authorizer: cognitoAuthorizer });
        projectAttachmentIdResource.addMethod('OPTIONS', corsIntegration);

        // Project users
        const projectUsersResource = projectIdResource.addResource('users');
        projectUsersResource.addMethod('GET', projectManagementIntegration, { authorizer: cognitoAuthorizer });
        projectUsersResource.addMethod('POST', projectManagementIntegration, { authorizer: cognitoAuthorizer });
        projectUsersResource.addMethod('OPTIONS', corsIntegration);

        const projectUserAssignmentIdResource = projectUsersResource.addResource('{assignmentId}');
        projectUserAssignmentIdResource.addMethod('PUT', projectManagementIntegration, { authorizer: cognitoAuthorizer });
        projectUserAssignmentIdResource.addMethod('DELETE', projectManagementIntegration, { authorizer: cognitoAuthorizer });
        projectUserAssignmentIdResource.addMethod('OPTIONS', corsIntegration);

        return projectsApi;
    }

    private createIssuesApi(props: MultiApiSubStackProps, cognitoConstruct: CognitoConstruct, corsIntegration: LambdaIntegration): RestApi {
        const issuesApi = new RestApi(this, "IssuesAPI", {
            restApiName: `${props.options.apiName}-Issues`,
            description: "Issue Management API",
            deployOptions: {
                stageName: props.options.apiStageName,
            },
        });

        // Create Cognito authorizer for this API
        const cognitoAuthorizer = new CognitoUserPoolsAuthorizer(this, 'IssuesApiCognitoAuthorizer', {
            cognitoUserPools: [cognitoConstruct.userPool],
            authorizerName: 'IssuesApiCognitoAuthorizer'
        });

        const issueManagementIntegration = new LambdaIntegration(this.lambdaConstruct.issueManagementLambda);

        // Issues endpoints
        const issuesResource = issuesApi.root.addResource('issues');
        issuesResource.addMethod('POST', issueManagementIntegration, { authorizer: cognitoAuthorizer });
        issuesResource.addMethod('OPTIONS', corsIntegration);

        const issueIdResource = issuesResource.addResource('{issueId}');
        issueIdResource.addMethod('GET', issueManagementIntegration, { authorizer: cognitoAuthorizer });
        issueIdResource.addMethod('PUT', issueManagementIntegration, { authorizer: cognitoAuthorizer });
        issueIdResource.addMethod('DELETE', issueManagementIntegration, { authorizer: cognitoAuthorizer });
        issueIdResource.addMethod('OPTIONS', corsIntegration);

        const issueStatusResource = issueIdResource.addResource('status');
        issueStatusResource.addMethod('PATCH', issueManagementIntegration, { authorizer: cognitoAuthorizer });
        issueStatusResource.addMethod('OPTIONS', corsIntegration);

        // Project-specific issues endpoints
        const projectsResource = issuesApi.root.addResource('projects');
        const projectIdResource = projectsResource.addResource('{projectId}');
        const projectIssuesResource = projectIdResource.addResource('issues');
        projectIssuesResource.addMethod('GET', issueManagementIntegration, { authorizer: cognitoAuthorizer });
        projectIssuesResource.addMethod('POST', issueManagementIntegration, { authorizer: cognitoAuthorizer });
        projectIssuesResource.addMethod('OPTIONS', corsIntegration);

        return issuesApi;
    }

    private createRfisApi(props: MultiApiSubStackProps, cognitoConstruct: CognitoConstruct, corsIntegration: LambdaIntegration): RestApi {
        const rfisApi = new RestApi(this, "RFIsAPI", {
            restApiName: `${props.options.apiName}-RFIs`,
            description: "RFI Management API",
            deployOptions: {
                stageName: props.options.apiStageName,
            },
        });

        // Create Cognito authorizer for this API
        const cognitoAuthorizer = new CognitoUserPoolsAuthorizer(this, 'RfisApiCognitoAuthorizer', {
            cognitoUserPools: [cognitoConstruct.userPool],
            authorizerName: 'RfisApiCognitoAuthorizer'
        });

        const rfiManagementIntegration = new LambdaIntegration(this.lambdaConstruct.rfiManagementLambda);

        // RFI endpoints
        const rfisResource = rfisApi.root.addResource('rfis');
        rfisResource.addMethod('POST', rfiManagementIntegration, { authorizer: cognitoAuthorizer });
        rfisResource.addMethod('OPTIONS', corsIntegration);

        const rfiIdResource = rfisResource.addResource('{rfiId}');
        rfiIdResource.addMethod('GET', rfiManagementIntegration, { authorizer: cognitoAuthorizer });
        rfiIdResource.addMethod('PUT', rfiManagementIntegration, { authorizer: cognitoAuthorizer });
        rfiIdResource.addMethod('DELETE', rfiManagementIntegration, { authorizer: cognitoAuthorizer });
        rfiIdResource.addMethod('OPTIONS', corsIntegration);

        // RFI workflow endpoints
        const rfiStatusResource = rfiIdResource.addResource('status');
        rfiStatusResource.addMethod('PATCH', rfiManagementIntegration, { authorizer: cognitoAuthorizer });
        rfiStatusResource.addMethod('OPTIONS', corsIntegration);

        const rfiSubmitResource = rfiIdResource.addResource('submit');
        rfiSubmitResource.addMethod('POST', rfiManagementIntegration, { authorizer: cognitoAuthorizer });
        rfiSubmitResource.addMethod('OPTIONS', corsIntegration);

        const rfiRespondResource = rfiIdResource.addResource('respond');
        rfiRespondResource.addMethod('POST', rfiManagementIntegration, { authorizer: cognitoAuthorizer });
        rfiRespondResource.addMethod('OPTIONS', corsIntegration);

        const rfiApproveResource = rfiIdResource.addResource('approve');
        rfiApproveResource.addMethod('POST', rfiManagementIntegration, { authorizer: cognitoAuthorizer });
        rfiApproveResource.addMethod('OPTIONS', corsIntegration);

        const rfiRejectResource = rfiIdResource.addResource('reject');
        rfiRejectResource.addMethod('POST', rfiManagementIntegration, { authorizer: cognitoAuthorizer });
        rfiRejectResource.addMethod('OPTIONS', corsIntegration);

        // RFI attachments
        const rfiAttachmentsResource = rfiIdResource.addResource('attachments');
        rfiAttachmentsResource.addMethod('GET', rfiManagementIntegration, { authorizer: cognitoAuthorizer });
        rfiAttachmentsResource.addMethod('POST', rfiManagementIntegration, { authorizer: cognitoAuthorizer });
        rfiAttachmentsResource.addMethod('OPTIONS', corsIntegration);

        const rfiAttachmentIdResource = rfiAttachmentsResource.addResource('{attachmentId}');
        rfiAttachmentIdResource.addMethod('GET', rfiManagementIntegration, { authorizer: cognitoAuthorizer });
        rfiAttachmentIdResource.addMethod('DELETE', rfiManagementIntegration, { authorizer: cognitoAuthorizer });
        rfiAttachmentIdResource.addMethod('OPTIONS', corsIntegration);

        // RFI comments
        const rfiCommentsResource = rfiIdResource.addResource('comments');
        rfiCommentsResource.addMethod('GET', rfiManagementIntegration, { authorizer: cognitoAuthorizer });
        rfiCommentsResource.addMethod('POST', rfiManagementIntegration, { authorizer: cognitoAuthorizer });
        rfiCommentsResource.addMethod('OPTIONS', corsIntegration);

        // Project-specific RFI endpoints
        const projectsResource = rfisApi.root.addResource('projects');
        const projectIdResource = projectsResource.addResource('{projectId}');
        const projectRFIsResource = projectIdResource.addResource('rfis');
        projectRFIsResource.addMethod('GET', rfiManagementIntegration, { authorizer: cognitoAuthorizer });
        projectRFIsResource.addMethod('POST', rfiManagementIntegration, { authorizer: cognitoAuthorizer });
        projectRFIsResource.addMethod('OPTIONS', corsIntegration);

        return rfisApi;
    }

    private setupDomainMappings(props: MultiApiSubStackProps): void {
        // Skip domain for LOCAL
        if (props.stageEnvironment === StageEnvironment.LOCAL) {
            return;
        }

        const stageOptions = findStageOptions(props.options, props.stageEnvironment);

        // Only create domain mapping if domain configuration is provided
        if (!stageOptions || !stageOptions.domainName || stageOptions.domainName.trim() === "") {
            console.log("No domain configuration found for stage, skipping domain mapping");
            return;
        }

        const domainName = DomainName.fromDomainNameAttributes(this, "APIDomainName", {
            domainName: stageOptions.domainName,
            domainNameAliasTarget: stageOptions.domainNameAliasTarget,
            domainNameAliasHostedZoneId: stageOptions.domainNameAliasHostedZoneId,
        });

        // Create separate base path mappings for each API
        new BasePathMapping(this, "IamApiBasePathMapping", {
            domainName: domainName,
            restApi: this.iamApi,
            basePath: "iam",
            stage: this.iamApi.deploymentStage,
        });

        new BasePathMapping(this, "ProjectsApiBasePathMapping", {
            domainName: domainName,
            restApi: this.projectsApi,
            basePath: "projects",
            stage: this.projectsApi.deploymentStage,
        });

        new BasePathMapping(this, "IssuesApiBasePathMapping", {
            domainName: domainName,
            restApi: this.issuesApi,
            basePath: "issues",
            stage: this.issuesApi.deploymentStage,
        });

        new BasePathMapping(this, "RfisApiBasePathMapping", {
            domainName: domainName,
            restApi: this.rfisApi,
            basePath: "rfis",
            stage: this.rfisApi.deploymentStage,
        });
    }

    // Getters for external access
    get corsLambdaArn(): string {
        return this.lambdaConstruct.corsLambdaArn;
    }

    get iamApiId(): string {
        return this.iamApi.restApiId;
    }

    get projectsApiId(): string {
        return this.projectsApi.restApiId;
    }

    get issuesApiId(): string {
        return this.issuesApi.restApiId;
    }

    get rfisApiId(): string {
        return this.rfisApi.restApiId;
    }
}