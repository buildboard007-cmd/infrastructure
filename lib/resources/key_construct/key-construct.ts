import {Fn} from "aws-cdk-lib";
import {Construct} from "constructs";
import {IKey, Key} from "aws-cdk-lib/aws-kms";
import {StageEnvironment} from "../../types/stage-environment";

interface KeyConstructProps {
    stageEnvironment: StageEnvironment;
}

export class KeyConstruct extends Construct {
    private readonly _dataKey: IKey;
    private readonly _snsSqsKey: IKey;

    constructor(scope: Construct, id: string, props: KeyConstructProps) {
        super(scope, id);

        if (props.stageEnvironment !== StageEnvironment.LOCAL) {
            this._dataKey = new Key(this, `DataKey`);
            this._dataKey.addAlias('alias/account-data-kmskey');

            this._snsSqsKey = new Key(this, `SnsSqsKey`);
            this._dataKey.addAlias('alias/account-sns-sqs-kmskey');
        } else {
            this._dataKey = new Key(this, 'DataKey');
            this._snsSqsKey = new Key(this, 'SnsSqsKey');
        }
    }

    get dataKey(): IKey {
        return this._dataKey;
    }

    get snsSqsKey(): IKey {
        return this._snsSqsKey;
    }
}
