import {StackOptions} from "./stack-options";
import {StageEnvironment} from "./stage-environment";
import {SharedLambdaSecurityGroup} from "../resources/security_construct/shared-lambda-security";
import * as ec2 from "aws-cdk-lib/aws-ec2";

export interface FuncProps {
    options: StackOptions;
    stageEnvironment: StageEnvironment;
    // Optional VPC config for database-accessing Lambdas
    vpcConfig?: {
        vpc: ec2.IVpc;
        databaseSecurityGroup: ec2.ISecurityGroup;
        vpcSubnets: ec2.SubnetSelection;
        allowPublicSubnet: boolean;
    };
    sharedLambdaSG?: SharedLambdaSecurityGroup;
}
