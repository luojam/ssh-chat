#!/usr/bin/env node
import * as cdk from 'aws-cdk-lib';
import { SshChatStack } from '../lib/ssh-chat-stack';

const app = new cdk.App();

new SshChatStack(app, 'SshChatStack', {
    env: {
        account: process.env.CDK_DEFAULT_ACCOUNT,
        region: process.env.CDK_DEFAULT_REGION,
    },
});
