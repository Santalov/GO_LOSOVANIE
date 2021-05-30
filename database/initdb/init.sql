create table block
(
    blockId      serial primary key,
    height       integer      not null,
    blockHash    bytea unique not null, -- to select by hash
    prevBlockId  integer      null references block (blockId) on delete no action on update no action,
    merkleTree   bytea        not null,
    proposerPkey bytea        not null,
    timestamp    bigint       not null
);

create table transaction
(
    txId                serial primary key,
    blockId             integer      not null references block (blockId) on delete cascade on update no action,
    index               integer      not null, -- to persist order of transactions in block
    txHash              bytea unique not null, -- to select by hash
    hashLink            bytea        null,
    valueType           bytea        null,
    voteType            integer,
    duration            integer      null,
    senderEphemeralPkey bytea        null,
    votersSumPkey       bytea        null,
    signature           bytea        not null
);

create table input
(
    txId        integer not null references transaction (txId) on delete cascade on update no action,
    index       integer not null, -- index in input array of transaction
    -- coinbase transactions have no input, so columns below are really not null
    prevTxId    integer not null references transaction (txId),
    outputIndex integer not null,
    primary key (txId, index)
);

create table output
(
    txId              integer not null references transaction (txId) on delete cascade on update no action,
    index             integer not null, -- index in input array of transaction
    value             integer not null,
    receiverSpendPkey bytea   not null,
    receiverScanPkey  bytea   null,
    isSpentByTx       integer null references transaction (txId),
    primary key (txId, index)
);

-- prohibit updates (blockchain must only be extended, which means insert,
-- or rewritten, which means delete incorrect values and insert correct)

create function prohibitUpdate()
    returns trigger as
$$
begin
    raise exception 'table % must never be updated', TG_RELNAME;
    return null;
end
$$ language plpgsql;

create trigger block_prohibitUpdate
    before update
    on block
execute function prohibitUpdate();

create trigger transaction_prohibitUpdate
    before update
    on transaction
execute function prohibitUpdate();

-- in outputs only column isSpentByTx might be updated

create function prohibitUpdateOutput()
    returns trigger as
$$
begin
    if (
            old.txId != new.txId or
            old.index != new.index or
            old.value != new.value or
            old.receiverSpendPkey != new.receiverSpendPkey or
            old.receiverScanPkey != new.receiverScanPkey
        ) then
        raise exception 'illegal column update in output table, only updating isSepntByTx is allowed';
        return null;
    else
        return new;
    end if;
end
$$ language plpgsql;

create trigger output_prohibitUpdate
    before update
    on output
execute function prohibitUpdateOutput();

-- triggers for block table
create function checkChainInsert()
    returns trigger as
$$
begin
    if (new.prevBlockId is null) then
        if ((select count(*) from block) != 1) then
            raise exception 'prevBlockId must be not null, null is possible only in first block';
        end if;
    else
        if ((select count(*) from block where prevBlockId = new.prevBlockId) > 1) then
            raise exception 'too many blocks have same prevBlockHash %', new.prevBlockId;
        end if;
    end if;
    return null;
end
$$ language plpgsql;

create trigger block_checkChainInsert
    after insert
    on block
    for each row
execute function checkChainInsert();


-- triggers for input and output table
create function updateOutputSpendingsInsert()
    returns trigger as
$$
begin
    update output set isSpentByTx = new.txid where output.txId = new.prevTxId and output.index = new.outputIndex;
    return null;
end
$$ language plpgsql;

create trigger input_upateOutputSpendingsInsert
    after insert
    on input
    for each row
execute function updateOutputSpendingsInsert();


create function updateOutputSpendingsDelete()
    returns trigger as
$$
begin
    update output set isSpentByTx = null where output.txId = old.prevTxId and output.index = old.outputIndex;
end
$$ language plpgsql;

create trigger input_updateOutputSpendingsDelete
    after delete
    on input
    for each row
execute function updateOutputSpendingsDelete();
