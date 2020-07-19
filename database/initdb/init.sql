create table block (
    blockHash bytea primary key,
    prevBlockHash bytea null references block(blockHash) on delete no action on update no action,
    merkleTree bytea not null,
    timestamp double precision not null
);

create table transaction (
    txId bytea primary key,
    index integer not null, -- to persist order of transactions in block
    typeValue bytea null,
    typeVote integer null,
    duration integer null,
    hashLink bytea null,
    signature bytea not null,
    blockHash bytea not null references block(blockHash) on delete cascade on update no action
);

create table input (
    txId bytea not null references transaction(txId) on delete cascade on update no action,
    index integer not null, -- index in input array of transaction
    -- coinbase transactions have no input, so columns below are really not null
    prevTxId bytea not null references transaction(txId),
    outputIndex integer not null,
    primary key(txId, index)
);

create table output (
    txId bytea not null references transaction(txId) on delete cascade on update no action,
    index integer not null, -- index in input array of transaction
    value integer not null,
    publicKeyTo bytea not null,
    isSpentByTx bytea null,
    primary key(txId, index)
);

-- prohibit updates (blockchain must only be extended, which means insert,
-- or rewritten, which means delete incorrect values and insert correct)

create function prohibitUpdate()
    returns trigger as $$
begin
    raise exception 'table % must never be updated', TG_RELNAME;
    return null;
end
$$ language plpgsql;

create trigger block_prohibitUpdate
    before update on block
    execute function prohibitUpdate();

create trigger transaction_prohibitUpdate
    before update on transaction
    execute function prohibitUpdate();

-- in outputs only column isSpentByTx might be updated

create function prohibitUpdateOutput()
    returns trigger as $$
begin
    if (old.index != new.index or old.value != new.value or old.publicKeyTo != new.publicKeyTo) then
        raise exception 'illegal column update in output table, only updating isSepntByTx is allowed';
        return null;
    else
        return new;
    end if;
end
$$ language plpgsql;

create trigger output_prohibitUpdate
    before update on output
    execute function prohibitUpdateOutput();

-- triggers for block table
create function checkChainInsert()
    returns trigger as $$
begin
    if (new.prevBlockHash is null) then
        if ((select count(*) from block) != 1) then
            raise exception 'prevBlockHash must be not null, null is possible only in first block';
        end if;
    else
        if ((select count(*) from block where prevBlockHash = new.prevBlockHash) > 1) then
            raise exception 'too many blocks have same prevBlockHash %', new.prevBlockHash;
        end if;
    end if;
    return null;
end
$$ language plpgsql;

create trigger block_checkChainInsert
    after insert on block
    for each row
    execute function checkChainInsert();


-- triggers for input and output table
create function updateOutputSpendingsInsert()
    returns trigger as $$
begin
    update output set isSpentByTx = new.txid where output.txId = new.prevTxId and output.index = new.outputIndex;
    return null;
end
$$ language plpgsql;

create trigger input_upateOutputSpendingsInsert
    after insert on input
    for each row
    execute function updateOutputSpendingsInsert();


create function updateOutputSpendingsDelete()
    returns trigger as $$
begin
    update output set isSpentByTx = null where output.txId = old.prevTxId and output.index = old.outputIndex;
end
$$ language plpgsql;

create trigger input_updateOutputSpendingsDelete
    after delete on input
    for each row
    execute function updateOutputSpendingsDelete();
