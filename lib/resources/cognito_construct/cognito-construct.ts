import {Construct} from "constructs";
import {CfnOutput, Duration, RemovalPolicy, Stack,} from "aws-cdk-lib";
import {
    AccountRecovery,
    BooleanAttribute,
    CfnUserPool,
    Mfa,
    OAuthScope,
    UserPool,
    UserPoolClient,
    UserPoolClientIdentityProvider,
    UserPoolDomain,
    UserPoolEmail,
    VerificationEmailStyle,
} from "aws-cdk-lib/aws-cognito";
import {ServicePrincipal} from "aws-cdk-lib/aws-iam";
import {CognitoConstructProps} from "../../types/cognito-construct-props";
import {StageEnvironment} from "../../types/stage-environment";

export class CognitoConstruct extends Construct {
    public readonly userPoolName: string;
    public readonly userPool: UserPool;
    public readonly userPoolClient: UserPoolClient;
    public readonly userPoolDomain: UserPoolDomain;

    constructor(scope: Construct, id: string, props: CognitoConstructProps) {
        super(scope, id);
        this.userPoolName = "Users";
        const isProd = props.stage === StageEnvironment.PROD;

        // MINIMAL User Pool - just email for authentication
        // Everything else comes from your database via Lambda
        this.userPool = new UserPool(this, "UserPool", {
            userPoolName: this.userPoolName,

            // Email-only authentication (simplest and most common)
            signInAliases: {
                email: true,
                username: false,
                phone: false,
            },
            signInCaseSensitive: false,

            // Allow self sign-up
            selfSignUpEnabled: true,
            autoVerify: {email: true},

            // Basic required attributes for user signup processing
            // Additional user data will be stored in the IAM database
            standardAttributes: {
                email: {
                    required: true,     // Required for authentication and organization assignment
                    mutable: true,
                }
            },

            // Custom attributes for user role identification
            customAttributes: {
                isSuperAdmin: new BooleanAttribute({
                    mutable: true
                })
            },

            // Basic password policy (adjust based on your needs)
            passwordPolicy: {
                minLength: 6,
                requireLowercase: true,
                requireUppercase: true,
                requireDigits: true,
                requireSymbols: false, // Optional - makes it easier for users
                tempPasswordValidity: Duration.days(7),
            },

            // Simple account recovery
            accountRecovery: AccountRecovery.EMAIL_ONLY,

            // MFA - Optional to reduce friction (you can enforce in app logic)
            mfa: Mfa.OFF, // Change to OPTIONAL if you want users to opt-in

            // Use Cognito's default email service (free tier included)
            email: UserPoolEmail.withCognito(),

            // Simple email verification
            userVerification: {
                emailSubject: "Verify your email",
                emailBody: "Your verification code is {####}",
                emailStyle: VerificationEmailStyle.CODE,
            },

            // Admin invite message template for user creation
            userInvitation: {
                emailSubject: "Welcome to BuildBoard - Set Up Your Account",
                emailBody: "Welcome to BuildBoard! Your account has been created. Please use the credentials below to log in and set up your password: Email: {username} Temporary Password: {####} You'll be prompted to create a new password on your first login. Best regards, The BuildBoard Team",
            },

            // Lambda triggers will be configured manually for V2.0 support

            // Deletion protection based on environment
            removalPolicy: isProd && !props.options.isTemporaryStack
                ? RemovalPolicy.RETAIN
                : RemovalPolicy.DESTROY,

            // No advanced features that cost extra
            // advancedSecurityMode: OFF by default
            // deviceTracking: OFF by default
            // No SMS MFA (costs money)
            // No custom SMS/Email senders (costs money)
        });

        // Configure Lambda triggers using CFN overrides for V2.0 support
        const cfnUserPool = this.userPool.node.defaultChild as CfnUserPool;

        // Pre-Token Generation V2.0 trigger for JWT token customization
        cfnUserPool.addPropertyOverride("LambdaConfig.PreTokenGenerationConfig", {
            LambdaArn: props.tokenCustomizerLambda.functionArn,
            LambdaVersion: "V2_0",  // V2.0 required for enhanced claim customization
        });

        // Post-Confirmation trigger for user signup processing
        cfnUserPool.addPropertyOverride("LambdaConfig.PostConfirmation",
            props.userSignupLambda.functionArn
        );

        // Grant Token Customizer Lambda permission to be invoked by Cognito
        props.tokenCustomizerLambda.addPermission("TokenCustomizerPermission", {
            action: "lambda:InvokeFunction",
            principal: new ServicePrincipal("cognito-idp.amazonaws.com"),
            sourceArn: this.userPool.userPoolArn,
        });

        // Grant User Signup Lambda permission to be invoked by Cognito
        props.userSignupLambda.addPermission("UserSignupPermission", {
            action: "lambda:InvokeFunction",
            principal: new ServicePrincipal("cognito-idp.amazonaws.com"),
            sourceArn: this.userPool.userPoolArn,
        });

        // Create Hosted UI domain (free)
        // Domain prefix can only contain lowercase letters, numbers, and hyphens
        const domainPrefix = `${props.options.serviceName}-users-${props.stage.toLowerCase()}`;
        this.userPoolDomain = new UserPoolDomain(this, "UserPoolDomain", {
            userPool: this.userPool,
            cognitoDomain: {
                domainPrefix: domainPrefix,
            },
        });

        // Single app client with minimal configuration
        this.userPoolClient = new UserPoolClient(this, "WebAppClient", {
            userPool: this.userPool,
            userPoolClientName: `web-app-client`,

            // Basic auth flows
            authFlows: {
                userSrp: true,  // Secure Remote Password (recommended)
                custom: true,   // For custom auth if needed
                userPassword: true // TODO: check later if we can disable it
            },

            // OAuth for Hosted UI
            oAuth: {
                flows: {
                    authorizationCodeGrant: true,  // Standard OAuth flow
                    implicitCodeGrant: false,      // Deprecated
                },
                scopes: [
                    OAuthScope.EMAIL,
                    OAuthScope.OPENID,
                    OAuthScope.PROFILE,
                ],
                callbackUrls: props.options.callbackUrls[props.stage],
                logoutUrls: props.options.logoutUrls[props.stage],
            },

            // Identity providers
            supportedIdentityProviders: [
                UserPoolClientIdentityProvider.COGNITO,
            ],

            // Token validity (AWS limits: ID token 5min-1day, Access token 5min-1day, Refresh token 60min-10years)
            idTokenValidity: Duration.hours(24),     // 1 day (max allowed for ID token)
            accessTokenValidity: Duration.hours(24), // 1 day (max allowed for access token)
            refreshTokenValidity: Duration.days(30), // 30 days for refresh token

            // Basic security
            enableTokenRevocation: true,
            preventUserExistenceErrors: true,

            // No secret for SPA/mobile apps
            generateSecret: false,
        });

        // Outputs
        new CfnOutput(this, "UserPoolId", {
            value: this.userPool.userPoolId,
            description: "Cognito User Pool ID",
            exportName: `${props.options.githubRepo}-user-pool-id`,
        });

        new CfnOutput(this, "UserPoolClientId", {
            value: this.userPoolClient.userPoolClientId,
            description: "Cognito User Pool Client ID",
            exportName: `${props.options.githubRepo}-user-pool-client-id`
        });

        new CfnOutput(this, "HostedUIDomain", {
            value: `https://${this.userPoolDomain.domainName}.auth.${Stack.of(this).region}.amazoncognito.com`,
            description: "Cognito Hosted UI URL",
            exportName: `${props.options.githubRepo}-cognito-hosted-ui-url`,
        });
    }
}
