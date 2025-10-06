import * as cdk from "aws-cdk-lib";
import { Construct } from "constructs";
import * as s3 from "aws-cdk-lib/aws-s3";
import * as ssm from "aws-cdk-lib/aws-ssm";
import { StageEnvironment } from "../../types/stage-environment";
import { StackOptions } from "../../types/stack-options";

interface S3ConstructProps {
    stageEnvironment: StageEnvironment;
    options: StackOptions;
}

export class S3Construct extends Construct {
    public readonly attachmentBucket: s3.Bucket;

    constructor(scope: Construct, id: string, props: S3ConstructProps) {
        super(scope, id);

        const stage = props.stageEnvironment.toLowerCase();

        // Create S3 bucket for attachments
        this.attachmentBucket = new s3.Bucket(this, "AttachmentBucket", {
            bucketName: `buildboard-attachments-${stage}`,
            versioned: true,
            encryption: s3.BucketEncryption.S3_MANAGED,
            blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
            lifecycleRules: [
                {
                    id: "archive-old-versions",
                    enabled: true,
                    noncurrentVersionExpiration: cdk.Duration.days(90),
                    noncurrentVersionTransitions: [
                        {
                            storageClass: s3.StorageClass.INFREQUENT_ACCESS,
                            transitionAfter: cdk.Duration.days(30),
                        },
                        {
                            storageClass: s3.StorageClass.GLACIER,
                            transitionAfter: cdk.Duration.days(60),
                        },
                    ],
                },
            ],
            cors: [
                {
                    allowedHeaders: ["*"],
                    allowedMethods: [
                        s3.HttpMethods.GET,
                        s3.HttpMethods.PUT,
                        s3.HttpMethods.POST,
                        s3.HttpMethods.DELETE,
                        s3.HttpMethods.HEAD,
                    ],
                    allowedOrigins: ["*"], // In production, restrict this to your domain
                    exposedHeaders: ["ETag"],
                    maxAge: 3000,
                },
            ],
            removalPolicy: cdk.RemovalPolicy.RETAIN, // Keep bucket on stack deletion
        });

        // Store bucket name in SSM Parameter Store for Lambda functions
        new ssm.StringParameter(this, "AttachmentBucketNameParameter", {
            parameterName: `/infrastructure/${stage}/s3/attachment-bucket-name`,
            stringValue: this.attachmentBucket.bucketName,
            description: "S3 bucket name for storing construction management attachments",
        });

        // Store bucket ARN in SSM Parameter Store
        new ssm.StringParameter(this, "AttachmentBucketArnParameter", {
            parameterName: `/infrastructure/${stage}/s3/attachment-bucket-arn`,
            stringValue: this.attachmentBucket.bucketArn,
            description: "S3 bucket ARN for storing construction management attachments",
        });

        // Add tags
        cdk.Tags.of(this.attachmentBucket).add("Project", "BuildBoard");
        cdk.Tags.of(this.attachmentBucket).add("Environment", stage);
        cdk.Tags.of(this.attachmentBucket).add("Purpose", "AttachmentStorage");
    }
}