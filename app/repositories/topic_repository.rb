# typed: strict
# frozen_string_literal: true

class TopicRepository < ApplicationRepository
  sig do
    params(
      topic_record: TopicRecord,
      can_update: T.nilable(T::Boolean),
      can_create_page: T.nilable(T::Boolean)
    ).returns(Topic)
  end
  def to_model(topic_record:, can_update: nil, can_create_page: nil)
    Topic.new(
      database_id: topic_record.id,
      number: topic_record.number,
      name: topic_record.name,
      description: topic_record.description,
      visibility: TopicVisibility.deserialize(topic_record.visibility),
      can_update:,
      can_create_page:,
      space: SpaceRepository.new.to_model(space_record: topic_record.space_record.not_nil!)
    )
  end

  sig do
    params(
      space_record: SpaceRecord,
      current_user_record: T.nilable(UserRecord)
    ).returns(T::Array[Topic])
  end
  def find_topics_by_space(space_record:, current_user_record:)
    # スペースに参加しているトピックを取得し、最新のlast_page_modified_atでソート
    topic_records = space_record.topic_records
      .select("topics.*, MAX(member_records.last_page_modified_at) as max_last_modified")
      .joins(:member_records)
      .where(member_records: {space_id: space_record.id})
      .group("topics.id")
      .preload(:space_record)
      .order("max_last_modified DESC NULLS LAST, topics.id DESC")

    # N+1を避けるため、必要なデータを一括取得
    topic_ids = topic_records.pluck(:id)
    topic_permissions_map = build_topic_permissions_map(
      space_record:,
      topic_ids:,
      current_user_record:
    )

    # 権限情報を含めてモデルに変換
    topic_records.map do |topic_record|
      permissions = topic_permissions_map[topic_record.id] || {can_update: false, can_create_page: false}

      to_model(
        topic_record:,
        can_update: permissions[:can_update],
        can_create_page: permissions[:can_create_page]
      )
    end
  end

  sig do
    params(
      space_record: SpaceRecord,
      topic_ids: T::Array[Types::DatabaseId],
      current_user_record: T.nilable(UserRecord)
    ).returns(T::Hash[Types::DatabaseId, T::Hash[Symbol, T::Boolean]])
  end
  private def build_topic_permissions_map(space_record:, topic_ids:, current_user_record:)
    return {} unless current_user_record

    # ユーザーのスペースメンバー情報を取得
    space_member = SpaceMemberRecord.find_by(
      space_id: space_record.id,
      user_id: current_user_record.id
    )

    return {} unless space_member

    # トピックメンバー情報を一括取得
    topic_members = TopicMemberRecord
      .where(space_member_id: space_member.id, topic_id: topic_ids)
      .preload(:topic_record) # ポリシーで必要になる可能性があるため
      .index_by(&:topic_id)

    # 各トピックの権限情報をマッピング
    topic_ids.each_with_object({}) do |topic_id, map|
      topic_member = topic_members[topic_id]

      if topic_member
        # TopicPolicyFactoryを使用して適切なポリシークラスを取得
        policy = TopicPolicyFactory.build(
          user_record: current_user_record,
          space_member_record: space_member,
          topic_member_record: topic_member
        )

        # ポリシークラスのメソッドを直接使用
        can_update = policy.can_update_topic?(topic_record: topic_member.topic_record.not_nil!)
        can_create_page = policy.can_create_page?(topic_record: topic_member.topic_record.not_nil!)

        map[topic_id] = {
          can_update:,
          can_create_page:
        }
      else
        # トピックメンバーではない場合は権限なし
        map[topic_id] = {
          can_update: false,
          can_create_page: false
        }
      end
    end
  end
end
