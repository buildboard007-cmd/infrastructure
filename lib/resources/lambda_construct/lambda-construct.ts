import {Construct} from 'constructs';
import {FuncProps} from "../../types/func-props";
import {InfrastructureApiGatewayCors} from "../function_construct/infrastructure-api-gateway-cors";
import {LambdaConstructProps} from "../../types/lambda-construct-props";
import {InfrastructureTokenCustomizer} from "../function_construct/infrastructure-token-customizer";
import {InfrastructureUserSignup} from "../function_construct/infrastructure-user-signup";
import {GoFunction} from "@aws-cdk/aws-lambda-go-alpha";
import {InfrastructureOrganizationManagement} from "../function_construct/infrastructure-organization-management";

export class LambdaConstruct extends Construct {

    private readonly infrastructureApiGatewayCors: InfrastructureApiGatewayCors;
    private readonly infrastructureTokenCustomizer: InfrastructureTokenCustomizer;
    private readonly infrastructureUserSignup: InfrastructureUserSignup;
    private readonly infrastructureOrganizationManagement: InfrastructureOrganizationManagement;

    constructor(scope: Construct, id: string, props: LambdaConstructProps) {
        super(scope, id);

        const funcProps: FuncProps = {
            options: props.options,
            stageEnvironment: props.stageEnvironment,
            builder: props.builder
        };

        this.infrastructureApiGatewayCors = new InfrastructureApiGatewayCors(this, 'InfrastructureApiGatewayCors', funcProps);
        this.infrastructureTokenCustomizer = new InfrastructureTokenCustomizer(this, 'InfrastructureTokenCustomizer', funcProps);
        this.infrastructureUserSignup = new InfrastructureUserSignup(this, 'InfrastructureUserSignup', funcProps);
        this.infrastructureOrganizationManagement = new InfrastructureOrganizationManagement(this, 'InfrastructureOrganizationManagement', funcProps);
    }

    get corsLambda(): GoFunction {
        return this.infrastructureApiGatewayCors.function;
    }

    get corsLambdaArn(): string {
        return this.infrastructureApiGatewayCors.functionArn;
    }

    get tokenCustomizerLambda(): GoFunction {
        return this.infrastructureTokenCustomizer.function;
    }

    get tokenCustomizerLambdaArn(): string {
        return this.infrastructureTokenCustomizer.functionArn;
    }

    get userSignupLambda(): GoFunction {
        return this.infrastructureUserSignup.function;
    }

    get userSignupLambdaArn(): string {
        return this.infrastructureUserSignup.functionArn;
    }

    get organizationManagementLambda(): GoFunction {
        return this.infrastructureOrganizationManagement.function;
    }

    get organizationManagementLambdaArn(): string {
        return this.infrastructureOrganizationManagement.functionArn;
    }
}