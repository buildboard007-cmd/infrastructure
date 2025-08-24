import {Construct} from 'constructs';
import {FuncProps} from "../../types/func-props";
import {InfrastructureApiGatewayCors} from "../function_construct/infrastructure-api-gateway-cors";
import {LambdaConstructProps} from "../../types/lambda-construct-props";
import {InfrastructureTokenCustomizer} from "../function_construct/infrastructure-token-customizer";
import {InfrastructureUserSignup} from "../function_construct/infrastructure-user-signup";
import {GoFunction} from "@aws-cdk/aws-lambda-go-alpha";
import {SharedLambdaSecurityGroup} from "../security_construct/shared-lambda-security";
import {getDatabaseVpcConfig} from "../../utils/database-vpc-config";
import * as ec2 from "aws-cdk-lib/aws-ec2";

export class LambdaConstruct extends Construct {

    private readonly infrastructureApiGatewayCors: InfrastructureApiGatewayCors;
    private readonly infrastructureTokenCustomizer: InfrastructureTokenCustomizer;
    private readonly infrastructureUserSignup: InfrastructureUserSignup;
    private readonly sharedLambdaSG: SharedLambdaSecurityGroup;

    constructor(scope: Construct, id: string, props: LambdaConstructProps) {
        super(scope, id);

        // Create shared VPC configuration and security group ONCE for all database-accessing Lambdas
        const vpcConfig = getDatabaseVpcConfig(this, 'SharedVpcConfig');
        this.sharedLambdaSG = new SharedLambdaSecurityGroup(this, "SharedLambdaSG", vpcConfig.vpc);

        // Allow the shared Lambda security group to connect to RDS
        vpcConfig.databaseSecurityGroup.addIngressRule(
            this.sharedLambdaSG.securityGroup,
            ec2.Port.tcp(5432),
            "Allow all Lambdas with shared SG to connect to PostgreSQL"
        );

        const funcProps: FuncProps = {
            options: props.options,
            stageEnvironment: props.stageEnvironment
        };

        // Extended props for database-accessing Lambdas
        const databaseFuncProps = {
            ...funcProps,
            vpcConfig,
            sharedLambdaSG: this.sharedLambdaSG
        };

        this.infrastructureApiGatewayCors = new InfrastructureApiGatewayCors(this, 'InfrastructureApiGatewayCors', funcProps);
        this.infrastructureTokenCustomizer = new InfrastructureTokenCustomizer(this, 'InfrastructureTokenCustomizer', databaseFuncProps);
        this.infrastructureUserSignup = new InfrastructureUserSignup(this, 'InfrastructureUserSignup', databaseFuncProps);
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
}