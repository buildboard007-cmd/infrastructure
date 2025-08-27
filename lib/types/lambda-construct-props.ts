import {StackOptions} from "./stack-options";
import {StageEnvironment} from "./stage-environment";

export interface LambdaConstructProps {
    options: StackOptions;
    stageEnvironment: StageEnvironment;
}
