import {NestedStack, NestedStackProps} from "aws-cdk-lib";
import {findStageOption as findStageOptions, StackOptions} from "../../types/stack-options";
import {StageEnvironment} from "../../types/stage-environment";
import {Construct} from "constructs";
import {KeyConstruct} from "../key_construct/key-construct";
import {LambdaConstruct} from "../lambda_construct/lambda-construct";
import {LambdaConstructProps} from "../../types/lambda-construct-props";
import {CognitoConstruct} from "../cognito_construct/cognito-construct";
import {BasePathMapping, DomainName, RestApi, LambdaIntegration} from "aws-cdk-lib/aws-apigateway";
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

        // Create Lambda integrations
        const orgManagementIntegration = new LambdaIntegration(this.lambdaConstruct.organizationManagementLambda);
        const corsIntegration = new LambdaIntegration(this.lambdaConstruct.corsLambda);

        // Create /org resource
        const orgResource = this.api.root.addResource('org');
        orgResource.addMethod('GET', orgManagementIntegration);
        orgResource.addMethod('PUT', orgManagementIntegration);
        orgResource.addMethod('OPTIONS', corsIntegration);

        // Create /search resource  
        const searchResource = this.api.root.addResource('search');
        searchResource.addMethod('GET', orgManagementIntegration); // Reusing for now, can be changed later
        searchResource.addMethod('OPTIONS', corsIntegration);

        // Skip domain for LOCAL
        if (props.stageEnvironment != StageEnvironment.LOCAL) {
            const stageOptions = findStageOptions(
                props.options,
                props.stageEnvironment
            );

            // need to figure out how to fail a CDK synth
            if (!stageOptions) {
                console.log("Didn't find a domain for the stage");
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
