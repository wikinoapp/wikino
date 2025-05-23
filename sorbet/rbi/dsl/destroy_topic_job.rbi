# typed: true

# DO NOT EDIT MANUALLY
# This is an autogenerated file for dynamic methods in `DestroyTopicJob`.
# Please instead update this file by running `bin/tapioca dsl DestroyTopicJob`.


class DestroyTopicJob
  class << self
    sig do
      params(
        topic_record_id: ::String,
        block: T.nilable(T.proc.params(job: DestroyTopicJob).void)
      ).returns(T.any(DestroyTopicJob, FalseClass))
    end
    def perform_later(topic_record_id:, &block); end

    sig { params(topic_record_id: ::String).void }
    def perform_now(topic_record_id:); end
  end
end
