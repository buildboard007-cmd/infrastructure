#!/usr/bin/env node
import 'source-map-support/register';
import * as cdk from 'aws-cdk-lib';
import {options} from "../config/config";
import {PipelineStack} from "../lib/pipeline/pipeline-stack";

const app = new cdk.App();


new PipelineStack(app, `${options.stackName}`, {
    env: {
        account: options.toolsAccount,
        region: options.defaultRegion
    },
    options: options,
});
