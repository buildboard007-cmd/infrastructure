import {Construct} from "constructs";
import {FuncProps} from "../../types/func-props";
import {GoFunction} from "@aws-cdk/aws-lambda-go-alpha";
import * as path from 'path';
import {Duration} from "aws-cdk-lib";
import {GetRetentionDays} from "../../utils/lambda-utils";
import {ssmPolicy} from "../../utils/policy-utils";
import {GetAccountId} from "../../utils/account-utils";
import * as ec2 from "aws-cdk-lib/aws-ec2";
import {SharedLambdaSecurityGroup} from "../security_construct/shared-lambda-security";
import {getDatabaseVpcConfig} from "../../utils/database-vpc-config";
import {getBaseLambdaEnvironment} from "../../utils/lambda-environment";

export class InfrastructureTokenCustomizer extends Construct {

    private readonly func: GoFunction;
    private readonly sharedLambdaSG: SharedLambdaSecurityGroup;

    constructor(scope: Construct, id: string, props: FuncProps) {
        super(scope, id);

        const account = GetAccountId(props.stageEnvironment);
        const functionName = `${props?.options.githubRepo}-token-customizer`

        // Get centralized VPC configuration
        const vpcConfig = getDatabaseVpcConfig(this, 'TokenCustomizer');

        // Use shared Lambda security group for database access
        // This pattern allows us to scale to 100+ Lambdas without modifying RDS security group each time
        this.sharedLambdaSG = new SharedLambdaSecurityGroup(this, "SharedLambdaSG", vpcConfig.vpc);

        // Allow the shared Lambda security group to connect to RDS
        // CDK automatically handles duplicate rules - if this exact rule already exists, it won't duplicate it
        // This is safe to call multiple times across different Lambda constructs
        vpcConfig.databaseSecurityGroup.addIngressRule(
            this.sharedLambdaSG.securityGroup,
            ec2.Port.tcp(5432),
            "Allow all Lambdas with shared SG to connect to PostgreSQL"
        );

        this.func = new GoFunction(this, id, {
            entry: path.join(__dirname, `../../../src/infrastructure-token-customizer`),
            functionName: functionName,
            timeout: Duration.seconds(10),
            environment: getBaseLambdaEnvironment(props.stageEnvironment),
            logRetention: GetRetentionDays(props),
            bundling: {
                goBuildFlags: ['-ldflags "-s -w"'],
            },
            // Configure VPC access for database connectivity
            vpc: vpcConfig.vpc,
            vpcSubnets: vpcConfig.vpcSubnets,
            securityGroups: [this.sharedLambdaSG.securityGroup],
            allowPublicSubnet: vpcConfig.allowPublicSubnet,
        });

        this.func.addToRolePolicy(ssmPolicy());
    }

    get function(): GoFunction {
        return this.func
    }

    get functionArn(): string {
        return this.func.functionArn;
    }
}