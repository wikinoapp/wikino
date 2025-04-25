# typed: strict
# frozen_string_literal: true

module PolicyConcerns
  module SpaceContext
    extend T::Sig

    SpaceBasedRecord = T.type_alias { T.any(PageRecord, SpaceRecord, TopicRecord) }

    sig do
      params(
        record: SpaceBasedRecord,
        space_member_record: T.nilable(SpaceMemberRecord)
      ).void
    end
    def initialize(record:, space_member_record:)
      @record = record
      @space_member_record = space_member_record
    end

    sig { returns(SpaceBasedRecord) }
    attr_reader :record
    private :record

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
  end
end
