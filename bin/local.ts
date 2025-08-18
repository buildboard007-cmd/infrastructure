#!/usr/bin/env node
import 'source-map-support/register';
import * as cdk from 'aws-cdk-lib';
import {options} from "../config/config";
import {MainStack} from '../lib/main-stack';
import {StageEnvironment} from "../lib/types/stage-environment";

const app = new cdk.App();

new MainStack(app,
    `${options.stackName}-AppStage`,
    {
        options: options,
        stageEnvironment: StageEnvironment.LOCAL
    });