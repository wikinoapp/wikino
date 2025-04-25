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
        space_record: T.nilable(SpaceRecord),
        space_member_record: T.nilable(SpaceMemberRecord)
      ).void
    end
    def initialize(record:, user_record: nil, space_record: nil, space_member_record: nil)
      @record = record
      @user_record = user_record
      @space_record = space_record
      @space_member_record = space_member_record || find_space_member_record

      unless valid_argument?
        raise ArgumentError, "user_record and space_record must be the same as space_member_record"
      end
    end

    sig { returns(SpaceBasedRecord) }
    attr_reader :record
    private :record

    sig { returns(T.nilable(UserRecord)) }
    attr_reader :user_record
    private :user_record

    sig { returns(T.nilable(SpaceRecord)) }
    attr_reader :space_record
    private :space_record

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

    sig { returns(T.nilable(SpaceMemberRecord)) }
    private def find_space_member_record
      user_record&.active_space_member_records&.find_by(space_record:)
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
    private def valid_argument?
      user_record&.id == space_member_record&.user_id &&
        space_record&.id == space_member_record&.space_id
    end
  end
end
