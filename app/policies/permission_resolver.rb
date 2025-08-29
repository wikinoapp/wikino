# typed: strict
# frozen_string_literal: true

# Space-Topic 2層構造の権限を解決し、適切なPolicyインスタンスを返す
# 権限の優先順位:
# 1. Space Owner → Space内の全権限（全Topic含む）
# 2. Topic Admin → Topic内の全権限
# 3. Topic Member → Topic内の基本操作権限
# 4. Space Member → Space内の基本操作権限（Topic未参加）
# 5. Guest → 公開コンテンツのみ閲覧
class PermissionResolver
  extend T::Sig

  sig do
    params(
      user_record: T.nilable(UserRecord),
      space_record: T.nilable(SpaceRecord),
      topic_record: T.nilable(TopicRecord)
    ).void
  end
  def initialize(user_record:, space_record:, topic_record: nil)
    @user_record = user_record
    @space_record = space_record
    @topic_record = topic_record

    # SpaceMemberRecordを取得
    @space_member_record = T.let(
      if user_record && space_record
        user_record.space_member_records.find_by(space_record:)
      end,
      T.nilable(SpaceMemberRecord)
    )

    # TopicMemberRecordを取得
    @topic_member_record = T.let(
      if user_record && topic_record
        user_record.topic_member_records.find_by(topic_record:)
      end,
      T.nilable(TopicMemberRecord)
    )
  end

  sig { returns(T::Wikino::PolicyInstance) }
  def resolve
    # 1. Space Ownerが最優先
    if space_member_record&.role == SpaceMemberRole::Owner.serialize
      return SpaceOwnerPolicy.new(
        user_record: user_record.not_nil!,
        space_member_record: space_member_record.not_nil!
      )
    end

    # 2. Topic権限をチェック
    if topic_record && topic_member_record
      return build_topic_policy
    end

    # 3. Space権限をチェック
    if space_member_record
      return SpaceMemberPolicy.new(
        user_record: user_record.not_nil!,
        space_member_record: space_member_record.not_nil!
      )
    end

    # 4. ゲスト権限
    SpaceGuestPolicy.new(user_record:)
  end

  sig { returns(T::Wikino::PolicyInstance) }
  def resolve_for_topic
    # Topicが指定されていない場合はSpace権限で解決
    if topic_record.nil?
      return resolve
    end

    # Topic固有の権限解決
    resolve
  end

  sig { returns(T.nilable(UserRecord)) }
  attr_reader :user_record
  private :user_record

  sig { returns(T.nilable(SpaceRecord)) }
  attr_reader :space_record
  private :space_record

  sig { returns(T.nilable(TopicRecord)) }
  attr_reader :topic_record
  private :topic_record

  sig { returns(T.nilable(SpaceMemberRecord)) }
  attr_reader :space_member_record
  private :space_member_record

  sig { returns(T.nilable(TopicMemberRecord)) }
  attr_reader :topic_member_record
  private :topic_member_record

  sig { returns(T::Wikino::PolicyInstance) }
  private def build_topic_policy
    # Topic Adminの場合
    if topic_member_record&.role == TopicMemberRole::Admin.serialize
      # 現在のポリシー構造では、Topic AdminもSpaceOwnerPolicyと同じ権限を持つ
      # 将来的にTopicAdminPolicyを作成する場合はここで分岐
      SpaceOwnerPolicy.new(
        user_record: user_record.not_nil!,
        space_member_record: space_member_record.not_nil!
      )
    else
      # Topic Memberの場合
      SpaceMemberPolicy.new(
        user_record: user_record.not_nil!,
        space_member_record: space_member_record.not_nil!
      )
    end
  end
end
