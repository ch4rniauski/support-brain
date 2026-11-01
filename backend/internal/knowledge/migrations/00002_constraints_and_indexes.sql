-- +goose Up
CREATE UNIQUE INDEX cluster_members_qa_pair_id_key ON cluster_members (qa_pair_id);

CREATE UNIQUE INDEX disputed_qa_pairs_pending_qa_pair_id_key
    ON disputed_qa_pairs (qa_pair_id)
    WHERE status = 'pending';

-- +goose Down
DROP INDEX IF EXISTS disputed_qa_pairs_pending_qa_pair_id_key;
DROP INDEX IF EXISTS cluster_members_qa_pair_id_key;
