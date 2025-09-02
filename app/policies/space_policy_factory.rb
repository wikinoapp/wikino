# typed: strict
# frozen_string_literal: true

# ロールに応じた適切なSpace Policyクラスを生成するFactory
class SpacePolicyFactory
  extend T::Sig

  sig do
    params(
      user_record: T.nilable(UserRecord),
      space_member_record: T.nilable(SpaceMemberRecord)
    ).returns(T::Wikino::SpacePolicyInstance)
  end
  def self.build(user_record:, space_member_record: nil)
    # 非メンバーの場合
    if space_member_record.nil?
      return SpaceGuestPolicy.new(user_record:)
    end

    # space_member_recordが存在する場合、user_recordも必ず存在するはず
    raise ArgumentError, "user_record must not be nil when space_member_record is present" if user_record.nil?

    # ロールに応じたPolicyを返す
    case space_member_record.role
    when SpaceMemberRole::Owner.serialize
      SpaceOwnerPolicy.new(user_record:, space_member_record:)
    when SpaceMemberRole::Member.serialize
      SpaceMemberPolicy.new(user_record:, space_member_record:)
    else
      raise ArgumentError, "Unknown role: #{space_member_record.role}"
    end
  end
end
