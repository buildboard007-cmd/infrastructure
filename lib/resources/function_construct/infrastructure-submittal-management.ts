import {Construct} from 'constructs';
import {Duration} from "aws-cdk-lib";
import {GoFunction} from '@aws-cdk/aws-lambda-go-alpha';
import {FuncProps} from "../../types/func-props";
import {getBaseLambdaEnvironment} from "../../utils/lambda-environment";
import {ssmPolicy} from "../../utils/policy-utils";

export class InfrastructureSubmittalManagement extends Construct {

    public readonly function: GoFunction;
    public readonly functionArn: string;

    constructor(scope: Construct, id: string, props: FuncProps) {
        super(scope, id);

        this.function = new GoFunction(this, 'InfrastructureSubmittalManagement', {
            functionName: 'infrastructure-submittal-management',
            entry: './src/infrastructure-submittal-management',
            timeout: Duration.seconds(30),
            memorySize: 256,
            environment: getBaseLambdaEnvironment(props.stageEnvironment)
        });

        this.function.addToRolePolicy(ssmPolicy());

        this.functionArn = this.function.functionArn;
    }
}