import {GoFunction} from "@aws-cdk/aws-lambda-go-alpha";
import {Construct} from "constructs";
import {FuncProps} from "../../types/func-props";
import * as path from 'path';
import {Duration} from "aws-cdk-lib";
import {GetRetentionDays} from "../../utils/lambda-utils";
import {getBaseLambdaEnvironment} from "../../utils/lambda-environment";
import {ssmPolicy} from "../../utils/policy-utils";
import * as s3 from "aws-cdk-lib/aws-s3";
import * as iam from "aws-cdk-lib/aws-iam";

interface AttachmentFuncProps extends FuncProps {
    attachmentBucket: s3.Bucket;
}

export class InfrastructureAttachmentManagement extends Construct {
    private readonly func: GoFunction;

    constructor(scope: Construct, id: string, props: AttachmentFuncProps) {
        super(scope, id);

        const functionName = `${props?.options.githubRepo}-attachment-management`

        // Get base environment and add S3 bucket name
        const environment = {
            ...getBaseLambdaEnvironment(props.stageEnvironment),
            BUCKET_NAME: props.attachmentBucket.bucketName,
        };

        this.func = new GoFunction(this, id, {
            entry: path.join(__dirname, `../../../src/infrastructure-attachment-management`),
            functionName: functionName,
            timeout: Duration.seconds(30), // Longer timeout for S3 operations
            environment: environment,
            logRetention: GetRetentionDays(props),
            bundling: {
                goBuildFlags: ['-ldflags "-s -w"'],
            },
        });

        // Add SSM policy for accessing database parameters
        this.func.addToRolePolicy(ssmPolicy());

        // Grant S3 permissions for the attachment bucket
        props.attachmentBucket.grantReadWrite(this.func);

        // Add additional S3 permissions for presigned URLs
        this.func.addToRolePolicy(new iam.PolicyStatement({
            effect: iam.Effect.ALLOW,
            actions: [
                "s3:GetObject",
                "s3:PutObject",
                "s3:DeleteObject",
                "s3:GetObjectVersion",
                "s3:PutObjectAcl",
                "s3:GetObjectAcl",
            ],
            resources: [
                props.attachmentBucket.bucketArn,
                `${props.attachmentBucket.bucketArn}/*`,
            ],
        }));

        // Add S3 list permissions for bucket operations
        this.func.addToRolePolicy(new iam.PolicyStatement({
            effect: iam.Effect.ALLOW,
            actions: [
                "s3:ListBucket",
                "s3:GetBucketLocation",
                "s3:GetBucketVersioning",
            ],
            resources: [props.attachmentBucket.bucketArn],
        }));
    }

    get function(): GoFunction {
        return this.func
    }

    get functionArn(): string {
        return this.func.functionArn;
    }
}