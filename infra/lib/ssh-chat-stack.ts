import { CfnOutput, IgnoreMode, RemovalPolicy, Size, Stack, StackProps, Tags } from 'aws-cdk-lib';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import * as ecrAssets from 'aws-cdk-lib/aws-ecr-assets';
import * as iam from 'aws-cdk-lib/aws-iam';
import { Construct } from 'constructs';
import * as path from 'path';
import { loadConfig } from './config';
import { configureSshChatInstance } from './user-data';

export class SshChatStack extends Stack {
    constructor(scope: Construct, id: string, props?: StackProps) {
        super(scope, id, props);

        // Add common tags to every resource created by this stack.
        Tags.of(this).add('Application', 'ssh-chat');
        Tags.of(this).add('ManagedBy', 'cdk');

        // Read deployment settings from CDK context.
        const config = loadConfig(this);

        // SSH Chat is intentionally reachable by users over TCP appPort, and there
        // are no private workloads that need NAT gateway egress in this deployment.
        // Keeping one public subnet avoids fixed NAT gateway cost for a single host.
        const vpc = new ec2.Vpc(this, 'Vpc', {
            maxAzs: 1,
            natGateways: 0,
            subnetConfiguration: [
                {
                    name: 'public',
                    subnetType: ec2.SubnetType.PUBLIC,
                    cidrMask: 24,
                },
            ],
        });

        // Limit inbound traffic to the configured app port and CIDR.
        const securityGroup = new ec2.SecurityGroup(this, 'InstanceSecurityGroup', {
            vpc,
            allowAllOutbound: true,
            description: 'Security group for the SSH Chat EC2 instance',
        });

        securityGroup.addIngressRule(
            ec2.Peer.ipv4(config.allowedAppCidr),
            ec2.Port.tcp(config.appPort),
            `Allow SSH Chat app traffic on TCP ${config.appPort}`,
        );

        // Let the instance use SSM and pull the app image from ECR.
        const instanceRole = new iam.Role(this, 'InstanceRole', {
            assumedBy: new iam.ServicePrincipal('ec2.amazonaws.com'),
            description: 'Role used by the SSH Chat EC2 instance for SSM and image pulls',
            managedPolicies: [
                iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonSSMManagedInstanceCore'),
            ],
        });

        // Build and publish the local Docker image as a CDK asset.
        const appImage = new ecrAssets.DockerImageAsset(this, 'AppImage', {
            directory: path.join(__dirname, '..', '..'),
            file: 'Dockerfile',
            // CDK stages Docker assets before invoking Docker, so .dockerignore
            // must protect both cdk.out and the final Docker build context.
            ignoreMode: IgnoreMode.DOCKER,
            platform: ecrAssets.Platform.LINUX_AMD64,
        });
        appImage.repository.grantPull(instanceRole);

        // Run the chat service on a small public Amazon Linux EC2 instance.
        const instance = new ec2.Instance(this, 'Instance', {
            vpc,
            vpcSubnets: {
                subnets: vpc.publicSubnets,
            },
            associatePublicIpAddress: true,
            instanceType: new ec2.InstanceType(config.instanceType),
            machineImage: ec2.MachineImage.latestAmazonLinux2023(),
            securityGroup,
            role: instanceRole,
            requireImdsv2: true,
            // App image URIs are written through user data, which only runs on first
            // boot. Replace the host on user-data changes; durable state stays on the
            // retained data volume and the endpoint stays on the Elastic IP.
            userDataCausesReplacement: true,
            blockDevices: [
                {
                    deviceName: '/dev/xvda',
                    volume: ec2.BlockDeviceVolume.ebs(20, {
                        encrypted: true,
                        volumeType: ec2.EbsDeviceVolumeType.GP3,
                    }),
                },
            ],
        });

        // This volume stores durable app state: the SQLite database and SSH host
        // key under /var/lib/ssh-chat. It is retained on stack deletion so chat
        // history and client host-key trust are not destroyed with infrastructure.
        const dataVolume = new ec2.Volume(this, 'DataVolume', {
            availabilityZone: vpc.publicSubnets[0].availabilityZone,
            size: Size.gibibytes(config.dataVolumeGiB),
            encrypted: true,
            volumeType: ec2.EbsDeviceVolumeType.GP3,
        });
        dataVolume.applyRemovalPolicy(RemovalPolicy.RETAIN);

        // Attach the retained data volume where user data will mount it.
        new ec2.CfnVolumeAttachment(this, 'DataVolumeAttachment', {
            device: '/dev/sdf',
            instanceId: instance.instanceId,
            volumeId: dataVolume.volumeId,
        });

        // Keep a stable public IP even if the instance is replaced.
        const elasticIp = new ec2.CfnEIP(this, 'ElasticIp', {
            domain: 'vpc',
        });

        new ec2.CfnEIPAssociation(this, 'ElasticIpAssociation', {
            allocationId: elasticIp.attrAllocationId,
            instanceId: instance.instanceId,
        });

        // Install Docker, mount storage, and start the app during first boot.
        configureSshChatInstance(instance, {
            appPort: config.appPort,
            imageUri: appImage.imageUri,
            region: Stack.of(this).region,
            dataVolume,
        });

        // Export the important values needed to operate or connect to the service.
        new CfnOutput(this, 'DataVolumeId', {
            value: dataVolume.volumeId,
            description: 'Retained EBS volume for /var/lib/ssh-chat',
        });

        new CfnOutput(this, 'AppImageUri', {
            value: appImage.imageUri,
            description: 'CDK-published Docker image URI',
        });

        new CfnOutput(this, 'ElasticIpAddress', {
            value: elasticIp.ref,
            description: 'Stable public IPv4 address for SSH Chat',
        });

        new CfnOutput(this, 'SshCommand', {
            value: `ssh -p ${config.appPort} ${elasticIp.ref}`,
            description: 'SSH command using the Elastic IP endpoint',
        });
    }
}
