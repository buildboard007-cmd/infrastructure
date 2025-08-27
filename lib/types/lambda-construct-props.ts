import {StackOptions} from "./stack-options";
import {StageEnvironment} from "./stage-environment";
import {OpenApiBuilder} from "openapi3-ts";

export interface LambdaConstructProps {
    options: StackOptions;
    stageEnvironment: StageEnvironment;
    builder: OpenApiBuilder;
}
