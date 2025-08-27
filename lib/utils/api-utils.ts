import {PathItemObject, ReferenceObject, SchemaObject,} from "openapi3-ts/dist/mjs/model/OpenApi";
import {OpenApiBuilder} from "openapi3-ts";

function isSchemaObject(object: SchemaObject | ReferenceObject) {
    return !Object.prototype.hasOwnProperty.call(object, "$ref");
}

const removeDiscriminatorFromObject = (
    object: SchemaObject | ReferenceObject
) => {
    if (!isSchemaObject(object)) {
        return;
    }
    const schemaObject = object as SchemaObject;
    delete schemaObject.discriminator;
    const properties = schemaObject.properties;
    if (properties) {
        for (const propertyObject of Object.values(properties)) {
            removeDiscriminatorFromObject(propertyObject);
        }
    }
    if (schemaObject.allOf instanceof Array) {
        for (const allOfObject of schemaObject.allOf) {
            removeDiscriminatorFromObject(allOfObject);
        }
    }
    if (schemaObject.anyOf instanceof Array) {
        for (const anyOfObject of schemaObject.anyOf) {
            removeDiscriminatorFromObject(anyOfObject);
        }
    }
};

export const removeDiscriminators = (builder: OpenApiBuilder) => {
    const schemas = builder.getSpec().components!.schemas;
    if (schemas) {
        for (const schema of Object.values(schemas)) {
            removeDiscriminatorFromObject(schema);
        }
    }
};

export const addSecuritySchemeExtension = (
    builder: OpenApiBuilder,
    region: string,
    account: string,
    userPoolArn: string
) => {
    // Use Cognito User Pool as the authorizer
    builder.getSpec().components!.securitySchemes = {};
    builder.addSecurityScheme("CognitoAuthorizer", {
        type: "apiKey",
        name: "Authorization",
        in: "header",
        "x-amazon-apigateway-authtype": "cognito_user_pools",
        "x-amazon-apigateway-authorizer": {
            type: "cognito_user_pools",
            providerARNs: [userPoolArn],
        },
    });
};

export const addLambdaExtension = (
    path: string,
    builder: OpenApiBuilder,
    region: string,
    account: string,
    funcName: string,
    httpMethod: "get" | "patch" | "post" | "put"
) => {
    let v;
    if (path !== '') {
        v = builder.getSpec().paths[path] as PathItemObject;
    } else {
        v = builder.getSpec().paths['/'] as PathItemObject;
    }
    if (!v || !v[httpMethod]) return;
    v[httpMethod]!["x-amazon-apigateway-integration"] = {
        httpMethod: "POST",
        uri: `arn:aws:apigateway:${region}:lambda:path/2015-03-31/functions/arn:aws:lambda:${region}:${account}:function:${funcName}/invocations`,
        passthroughBehavior: "when_no_match",
        contentHandling: "CONVERT_TO_TEXT",
        type: "aws_proxy",
    };
    // Security will be handled at Lambda function level
};

export const addCors = (
    path: string,
    builder: OpenApiBuilder,
    region: string,
    account: string,
    corsFunctionName: string
) => {
    let v = builder.getSpec().paths[path] as PathItemObject;
    if (!v) {
        console.error(
            `Error: Path ${path} not found in source spec: specs/source_spec.yaml.`
        );
        return;
    }
    if (v.options) return;

    v.options = {
        responses: {
            "200": {
                description: "Empty OPTIONS",
            },
        },
        "x-amazon-apigateway-integration": {
            httpMethod: "POST",
            uri: `arn:aws:apigateway:${region}:lambda:path/2015-03-31/functions/arn:aws:lambda:${region}:${account}:function:${corsFunctionName}/invocations`,
            passthroughBehavior: "when_no_match",
            type: "aws_proxy",
        },
    };
};
