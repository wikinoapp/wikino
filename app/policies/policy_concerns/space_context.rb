# typed: strict
# frozen_string_literal: true

module PolicyConcerns
  module SpaceContext
    extend T::Sig

    SpaceBasedRecord = T.type_alias { T.any(PageRecord, SpaceRecord, TopicRecord) }

    sig do
      params(
        record: SpaceBasedRecord,
        user_record: T.nilable(UserRecord),
        space_member_record: T.nilable(SpaceMemberRecord)
      ).void
    end
    def initialize(record:, user_record: nil, space_member_record: nil)
      @record = record
      @user_record = user_record
      @space_member_record = space_member_record

      if mismatched_relations?
        raise ArgumentError, [
          "Mismatched relations.",
          "user_record.id: #{user_record&.id.inspect}",
          "space_member_record.user_id: #{space_member_record&.user_id.inspect}"
        ].join(" ")
      end
    end

    sig { returns(SpaceBasedRecord) }
    attr_reader :record
    private :record

    sig { returns(T.nilable(UserRecord)) }
    attr_reader :user_record
    private :user_record

    sig { returns(T.nilable(SpaceMemberRecord)) }
    attr_reader :space_member_record
    private :space_member_record

    sig { returns(String) }
    private def space_id
      case record
      when SpaceRecord
        record.id
      else
        record.space_id
      end
    end

    sig { returns(T::Boolean) }
    private def same_space_member?
      space_member_record&.space_id == space_id
    end

    sig { returns(SpaceMemberRecord) }
    private def space_member_record!
      space_member_record.not_nil!
    end

    sig { returns(T::Boolean) }
    private def mismatched_relations?
      if !user_record.nil? && !space_member_record.nil?
        user_record.not_nil!.id != space_member_record.not_nil!.user_id
      else
        false
      end
    end
  end
end
