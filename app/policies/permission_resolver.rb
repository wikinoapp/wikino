# typed: strict
# frozen_string_literal: true

# Space権限を解決し、適切なSpacePolicyインスタンスを返す
# Topic権限が必要な場合は、返されたSpacePolicyの
# topic_policy_for(topic_record:)メソッドで取得する
class PermissionResolver
  extend T::Sig

  sig do
    params(
      user_record: T.nilable(UserRecord),
      space_record: T.nilable(SpaceRecord)
    ).void
  end
  def initialize(user_record:, space_record:)
    @user_record = user_record
    @space_record = space_record

    # SpaceMemberRecordを取得
    @space_member_record = T.let(
      if user_record && space_record
        user_record.space_member_records.find_by(space_record:)
      end,
      T.nilable(SpaceMemberRecord)
    )
  end

  sig { returns(T::Wikino::SpacePolicyInstance) }
  def resolve
    # 1. Space Ownerの場合
    if space_member_record&.role == SpaceMemberRole::Owner.serialize
      return SpaceOwnerPolicy.new(
        user_record: user_record.not_nil!,
        space_member_record: space_member_record.not_nil!
      )
    end

    # 2. Space Memberの場合
    if space_member_record
      return SpaceMemberPolicy.new(
        user_record: user_record.not_nil!,
        space_member_record: space_member_record.not_nil!
      )
    end

    # 3. ゲストの場合
    SpaceGuestPolicy.new(user_record:)
  end

  sig { returns(T.nilable(UserRecord)) }
  attr_reader :user_record
  private :user_record

  sig { returns(T.nilable(SpaceRecord)) }
  attr_reader :space_record
  private :space_record

  sig { returns(T.nilable(SpaceMemberRecord)) }
  attr_reader :space_member_record
  private :space_member_record
end
