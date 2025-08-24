import {StackOptions} from "./stack-options";
import {StageEnvironment} from "./stage-environment";
import {IFunction} from "aws-cdk-lib/aws-lambda";

/**
 * Props for the CognitoConstruct
 * 
 * Defines the required dependencies for setting up AWS Cognito User Pool
 * with Lambda triggers for token customization and user signup processing.
 */
export interface CognitoConstructProps {
    /** Stack configuration options including service name, URLs, and environment settings */
    options: StackOptions;
    
    /** Lambda function for Pre-Token Generation V2.0 trigger (adds custom claims to JWT tokens) */
    tokenCustomizerLambda: IFunction;
    
    /** Lambda function for Post-Confirmation trigger (processes new user signups) */
    userSignupLambda: IFunction;
    
    /** Deployment stage (DEV, STAGING, PROD) for environment-specific configuration */
    stage: StageEnvironment;
}