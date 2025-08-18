import {findStageOption, StackOptions} from "../types/stack-options";
import {StageEnvironment} from "../types/stage-environment";


export const findAccount = (options: StackOptions, env: StageEnvironment): string =>
    findStageOption(options, env).account;
