import {Construct} from 'constructs';
import {FuncProps} from "../../types/func-props";
import {InfrastructureApiGatewayCors} from "../function_construct/infrastructure-api-gateway-cors";
import {LambdaConstructProps} from "../../types/lambda-construct-props";
import {InfrastructureTokenCustomizer} from "../function_construct/infrastructure-token-customizer";
import {InfrastructureUserSignup} from "../function_construct/infrastructure-user-signup";
import {GoFunction} from "@aws-cdk/aws-lambda-go-alpha";
import {InfrastructureOrganizationManagement} from "../function_construct/infrastructure-organization-management";
import {InfrastructureLocationManagement} from "../function_construct/infrastructure-location-management";
import {InfrastructureRolesManagementFunction} from "../function_construct/infrastructure-roles-management";
import {InfrastructurePermissionsManagementFunction} from "../function_construct/infrastructure-permissions-management";

export class LambdaConstruct extends Construct {

    private readonly infrastructureApiGatewayCors: InfrastructureApiGatewayCors;
    private readonly infrastructureTokenCustomizer: InfrastructureTokenCustomizer;
    private readonly infrastructureUserSignup: InfrastructureUserSignup;
    private readonly infrastructureOrganizationManagement: InfrastructureOrganizationManagement;
    private readonly infrastructureLocationManagement: InfrastructureLocationManagement;
    private readonly infrastructureRolesManagement: InfrastructureRolesManagementFunction;
    private readonly infrastructurePermissionsManagement: InfrastructurePermissionsManagementFunction;

    constructor(scope: Construct, id: string, props: LambdaConstructProps) {
        super(scope, id);

        const funcProps: FuncProps = {
            options: props.options,
            stageEnvironment: props.stageEnvironment
        };

        this.infrastructureApiGatewayCors = new InfrastructureApiGatewayCors(this, 'InfrastructureApiGatewayCors', funcProps);
        this.infrastructureTokenCustomizer = new InfrastructureTokenCustomizer(this, 'InfrastructureTokenCustomizer', funcProps);
        this.infrastructureUserSignup = new InfrastructureUserSignup(this, 'InfrastructureUserSignup', funcProps);
        this.infrastructureOrganizationManagement = new InfrastructureOrganizationManagement(this, 'InfrastructureOrganizationManagement', funcProps);
        this.infrastructureLocationManagement = new InfrastructureLocationManagement(this, 'InfrastructureLocationManagement', funcProps);
        this.infrastructureRolesManagement = new InfrastructureRolesManagementFunction(this, 'InfrastructureRolesManagement', funcProps);
        this.infrastructurePermissionsManagement = new InfrastructurePermissionsManagementFunction(this, 'InfrastructurePermissionsManagement', funcProps);
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

    get organizationManagementLambda(): GoFunction {
        return this.infrastructureOrganizationManagement.function;
    }

    get organizationManagementLambdaArn(): string {
        return this.infrastructureOrganizationManagement.functionArn;
    }

    get locationManagementLambda(): GoFunction {
        return this.infrastructureLocationManagement.function;
    }

    get locationManagementLambdaArn(): string {
        return this.infrastructureLocationManagement.functionArn;
    }

    get rolesManagementLambda(): GoFunction {
        return this.infrastructureRolesManagement.function;
    }

    get rolesManagementLambdaArn(): string {
        return this.infrastructureRolesManagement.functionArn;
    }

    get permissionsManagementLambda(): GoFunction {
        return this.infrastructurePermissionsManagement.function;
    }

    get permissionsManagementLambdaArn(): string {
        return this.infrastructurePermissionsManagement.functionArn;
    }
}