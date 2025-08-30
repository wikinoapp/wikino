# typed: strict
# frozen_string_literal: true

# ロールに応じた適切なPolicyクラスを生成するFactory
class SpaceMemberPolicyFactory
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
