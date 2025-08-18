import {NestedStack, NestedStackProps} from "aws-cdk-lib";
import {StackOptions} from "../../types/stack-options";
import {StageEnvironment} from "../../types/stage-environment";
import {Construct} from "constructs";
import {KeyConstruct} from "../key_construct/key-construct";
import {IKey} from "aws-cdk-lib/aws-kms";
import {LambdaConstruct} from "../lambda_construct/lambda-construct";
import {LambdaConstructProps} from "../../types/lambda-construct-props";

interface KeyProps extends NestedStackProps{
    options: StackOptions;
    stageEnvironment: StageEnvironment;
}

export class SubStack extends NestedStack {
    private readonly keyConstruct: KeyConstruct;
    private readonly lambdaConstruct: LambdaConstruct;

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
