# AWS deployment Infrastructure

This directory contains the AWS CDK app for deploying SSH Chat.

## Commands

```sh
npm install
npm run build
npm run synth
npm run diff
npm run deploy
```

Docker must be running before `npm run deploy`, because CDK builds and publishes the application Docker image as a deployment asset.

Run `npx cdk bootstrap aws://ACCOUNT_ID/REGION` before the first deploy in an account and region. This stack uses CDK assets, including a Docker image asset that is published through bootstrap resources such as S3 and ECR.

The stack allocates an Elastic IP for a stable SSH endpoint.

The stack also creates a retained EBS data volume mounted at `/var/lib/ssh-chat`. It stores the SQLite database and SSH host key, so it is intentionally not deleted by `cdk destroy`.

## Deployment Semantics

Application image updates are deployed by replacing the EC2 instance. The CDK-published Docker image URI is written into first-boot user data, and the instance is configured with `userDataCausesReplacement: true` so a changed image URI creates a fresh host instead of leaving the running container on the old image.

Expect a brief service interruption while the replacement instance boots, mounts the retained `/var/lib/ssh-chat` EBS volume, pulls the new image, and starts `ssh-chat.service`. The retained EBS volume preserves the SQLite database and SSH host key, and the Elastic IP is re-associated to keep the public endpoint stable.

`npm run destroy` runs `cdk destroy`. Do not run it against production without making sure related data is secure. To remove the retained data as well, note the `DataVolumeId` stack output before destroying the stack, then manually delete that EBS volume in AWS after the stack is gone.

## DNS

Cloudflare `A` record `ssh.luja.dev` pointing at the `ElasticIpAddress` stack output. The app connection command is:

```sh
ssh -p 2222 ELASTIC_IP_OR_DOMAIN
```

## Runtime Checks

Use these from an AWS Systems Manager Session Manager shell on the instance:

```sh
sudo tail -n 200 /var/log/ssh-chat-user-data.log
sudo systemctl status ssh-chat --no-pager
sudo journalctl -u ssh-chat -n 200 --no-pager
sudo docker ps
sudo ss -ltnp
findmnt /var/lib/ssh-chat
ls -la /var/lib/ssh-chat
test -f /var/lib/ssh-chat/ssh-chat.sqlite && ls -lh /var/lib/ssh-chat/ssh-chat.sqlite
```
