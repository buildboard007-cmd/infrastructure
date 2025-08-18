import {Construct} from 'constructs';
import {FuncProps} from "../../types/func-props";
import {InfrastructureApiGatewayCors} from "../function_construct/infrastructure-api-gateway-cors";
import {LambdaConstructProps} from "../../types/lambda-construct-props";

export class LambdaConstruct extends Construct {
    
    private readonly infrastructureApiGatewayCors: InfrastructureApiGatewayCors;

    constructor(scope: Construct, id: string, props: LambdaConstructProps) {
        super(scope, id);

        const funcProps: FuncProps = {
            options: props.options,
            stageEnvironment: props.stageEnvironment
        };

        this.infrastructureApiGatewayCors = new InfrastructureApiGatewayCors(this, 'InfrastructureApiGatewayCors', funcProps);
    }

    get corsLambdaArn(): string {
        return this.infrastructureApiGatewayCors.function.functionArn;
    }
}