import { Stack } from 'aws-cdk-lib';

export interface SshChatConfig {
    readonly appPort: number;
    readonly allowedAppCidr: string;
    readonly instanceType: string;
    readonly dataVolumeGiB: number;
}

export function loadConfig(stack: Stack): SshChatConfig {
    const appPort = positiveInteger(contextValue(stack, 'appPort'), 'appPort');
    const allowedAppCidr = requiredString(contextValue(stack, 'allowedAppCidr'), 'allowedAppCidr');
    const instanceType = requiredString(contextValue(stack, 'instanceType'), 'instanceType');
    const dataVolumeGiB = positiveInteger(contextValue(stack, 'dataVolumeGiB'), 'dataVolumeGiB');

    if (appPort < 1 || appPort > 65535) {
        throw new Error(`appPort must be between 1 and 65535, got ${appPort}`);
    }

    if (!isIpv4Cidr(allowedAppCidr)) {
        throw new Error(`allowedAppCidr must be an IPv4 CIDR block, got ${allowedAppCidr}`);
    }

    if (dataVolumeGiB < 1) {
        throw new Error(`dataVolumeGiB must be at least 1, got ${dataVolumeGiB}`);
    }

    return {
        appPort,
        allowedAppCidr,
        instanceType,
        dataVolumeGiB,
    };
}

function contextValue(stack: Stack, key: string): unknown {
    return stack.node.tryGetContext(key);
}

function requiredString(value: unknown, key: string): string {
    if (typeof value !== 'string' || value.trim() === '') {
        throw new Error(`${key} must be a non-empty string`);
    }

    return value.trim();
}

function positiveInteger(value: unknown, key: string): number {
    const numberValue = typeof value === 'number' ? value : Number(value);

    if (!Number.isInteger(numberValue) || numberValue <= 0) {
        throw new Error(`${key} must be a positive integer, got ${value}`);
    }

    return numberValue;
}

function isIpv4Cidr(value: string): boolean {
    const match = value.match(/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})\/(\d{1,2})$/);
    if (!match) {
        return false;
    }

    const octets = match.slice(1, 5).map(Number);
    const prefix = Number(match[5]);

    return octets.every((octet) => octet >= 0 && octet <= 255) && prefix >= 0 && prefix <= 32;
}
