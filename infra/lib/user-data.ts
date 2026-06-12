import * as ec2 from 'aws-cdk-lib/aws-ec2';

export interface SshChatInstanceUserDataOptions {
    readonly appPort: number;
    readonly imageUri: string;
    readonly region: string;
    readonly dataVolume: ec2.IVolume;
}

export function configureSshChatInstance(
    instance: ec2.Instance,
    options: SshChatInstanceUserDataOptions,
): void {
    // Add each startup script in the order the instance needs to run it.
    addDataVolumeMountUserData(instance, options.dataVolume);
    addDockerInstallUserData(instance);
    addEcrImagePullUserData(instance, options.imageUri, options.region);
    addSshChatServiceUserData(instance, options.imageUri, options.appPort);
}

function addDockerInstallUserData(instance: ec2.Instance): void {
    // Install Docker if needed, then make sure it starts now and after reboots.
    instance.addUserData(
        'echo "Starting Docker setup"',
        'if ! command -v docker >/dev/null 2>&1; then',
        '  dnf install -y docker',
        'fi',
        'systemctl enable docker',
        'systemctl start docker',
        'docker --version',
        'echo "Docker setup complete"',
    );
}

function addEcrImagePullUserData(instance: ec2.Instance, imageUri: string, region: string): void {
    // Log in to the image's ECR registry and pull the container image once.
    instance.addUserData(
        'echo "Starting ECR image pull"',
        'if ! command -v aws >/dev/null 2>&1; then',
        '  dnf install -y awscli',
        'fi',
        `APP_IMAGE_URI='${imageUri}'`,
        `AWS_REGION='${region}'`,
        // Bash %% removes the longest match from the end; this keeps only the registry host.
        'ECR_REGISTRY="${APP_IMAGE_URI%%/*}"',
        'aws ecr get-login-password --region "$AWS_REGION" | docker login --username AWS --password-stdin "$ECR_REGISTRY"',
        'docker pull "$APP_IMAGE_URI"',
        'echo "ECR image pull complete"',
    );
}

function addSshChatServiceUserData(
    instance: ec2.Instance,
    imageUri: string,
    appPort: number,
): void {
    const serviceUnit = renderSshChatServiceUnit(appPort);

    // Write a systemd service so Linux keeps the ssh-chat container running.
    instance.addUserData(
        'echo "Writing ssh-chat systemd service"',
        // Heredocs write the following lines into a file until the marker appears again.
        'cat >/etc/ssh-chat.env <<SSH_CHAT_ENV',
        `APP_IMAGE_URI=${imageUri}`,
        'SSH_CHAT_ENV',
        'chmod 0644 /etc/ssh-chat.env',
        "cat >/etc/systemd/system/ssh-chat.service <<'SSH_CHAT_SERVICE'",
        ...serviceUnit.split('\n'),
        'SSH_CHAT_SERVICE',
        'systemctl daemon-reload',
        'systemctl enable ssh-chat',
        'systemctl restart ssh-chat',
        'echo "ssh-chat systemd service started"',
    );
}

function renderSshChatServiceUnit(appPort: number): string {
    // systemd runs this unit at boot and restarts the container if it exits.
    return `[Unit]
Description=SSH Chat container
Wants=network-online.target
After=network-online.target docker.service
Requires=docker.service
RequiresMountsFor=/var/lib/ssh-chat

[Service]
EnvironmentFile=/etc/ssh-chat.env
Restart=always
RestartSec=5
TimeoutStartSec=0
TimeoutStopSec=30
ExecStartPre=-/usr/bin/docker rm -f ssh-chat
ExecStart=/usr/bin/docker run --name ssh-chat --rm --pull never -p ${appPort}:${appPort} -e SSH_CHAT_HOST=0.0.0.0 -e SSH_CHAT_PORT=${appPort} -e SSH_CHAT_HOST_KEY_PATH=/var/lib/ssh-chat/ssh_host_ed25519 -e SSH_CHAT_SQLITE_PATH=/var/lib/ssh-chat/ssh-chat.sqlite -v /var/lib/ssh-chat:/var/lib/ssh-chat \${APP_IMAGE_URI}
ExecStop=/usr/bin/docker stop -t 20 ssh-chat

[Install]
WantedBy=multi-user.target`;
}

function addDataVolumeMountUserData(instance: ec2.Instance, dataVolume: ec2.IVolume): void {
    // Prepare the persistent EBS volume used for host keys and SQLite data.
    instance.addUserData(
        // Exit on errors and log all user-data output to both a file and the console.
        'set -euxo pipefail',
        'exec > >(tee -a /var/log/ssh-chat-user-data.log | logger -t ssh-chat-user-data -s 2>/dev/console) 2>&1',
        'echo "Starting ssh-chat data volume setup"',
        `DATA_VOLUME_ID='${dataVolume.volumeId}'`,
        'DATA_VOLUME_ID_NO_DASH="${DATA_VOLUME_ID//-/}"',
        'DATA_MOUNT_POINT=/var/lib/ssh-chat',
        'DATA_DEVICE=',
        // EBS device names can differ on Nitro instances, so try every expected path.
        'for attempt in $(seq 1 60); do',
        '  for candidate in "/dev/disk/by-id/nvme-Amazon_Elastic_Block_Store_${DATA_VOLUME_ID}" "/dev/disk/by-id/nvme-Amazon_Elastic_Block_Store_${DATA_VOLUME_ID_NO_DASH}" /dev/xvdf /dev/sdf; do',
        '    if [ -e "$candidate" ]; then',
        '      DATA_DEVICE="$(readlink -f "$candidate")"',
        '      echo "Found data volume device ${DATA_DEVICE} from ${candidate}"',
        '      break 2',
        '    fi',
        '  done',
        '  echo "Waiting for data volume ${DATA_VOLUME_ID} to appear, attempt ${attempt}"',
        '  sleep 2',
        'done',
        'if [ -z "$DATA_DEVICE" ]; then',
        '  echo "Data volume ${DATA_VOLUME_ID} did not appear"',
        '  lsblk',
        '  exit 1',
        'fi',
        // Only format brand-new disks; existing filesystems may contain app data.
        'if ! blkid "$DATA_DEVICE" >/dev/null 2>&1; then',
        '  echo "No filesystem detected on ${DATA_DEVICE}; creating ext4 filesystem"',
        '  mkfs.ext4 -F "$DATA_DEVICE"',
        'else',
        '  echo "Existing filesystem detected on ${DATA_DEVICE}; leaving data intact"',
        'fi',
        'DATA_UUID="$(blkid -s UUID -o value "$DATA_DEVICE")"',
        'if [ -z "$DATA_UUID" ]; then',
        '  echo "Unable to determine filesystem UUID for ${DATA_DEVICE}"',
        '  exit 1',
        'fi',
        'mkdir -p "$DATA_MOUNT_POINT"',
        // Mount by UUID so the volume keeps working if Linux renames the device.
        'if ! grep -q "UUID=${DATA_UUID} ${DATA_MOUNT_POINT} " /etc/fstab; then',
        '  echo "UUID=${DATA_UUID} ${DATA_MOUNT_POINT} ext4 defaults,nofail 0 2" >> /etc/fstab',
        'fi',
        'mountpoint -q "$DATA_MOUNT_POINT" || mount "$DATA_MOUNT_POINT"',
        'chown root:root "$DATA_MOUNT_POINT"',
        'chmod 0755 "$DATA_MOUNT_POINT"',
        'findmnt "$DATA_MOUNT_POINT"',
        'echo "ssh-chat data volume setup complete"',
    );
}
