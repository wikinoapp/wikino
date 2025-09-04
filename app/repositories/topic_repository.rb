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
    topic_records = TopicRecord
      .select("topics.*, MAX(member_records.last_page_modified_at) as max_last_modified")
      .joins(:member_records)
      .where(space_id: space_record.id)
      .where(member_records: {space_id: space_record.id})
      .group("topics.id")
      .preload(:space_record)
      .order("max_last_modified DESC NULLS LAST")

    # 権限情報を含めてモデルに変換
    topic_records.map do |topic_record|
      can_update = false
      can_create_page = false

      if current_user_record
        # 現在のユーザーがスペースメンバーか確認
        space_member = SpaceMemberRecord.find_by(
          space_id: space_record.id,
          user_id: current_user_record.id
        )

        if space_member
          # トピックメンバーか確認
          topic_member = TopicMemberRecord.find_by(
            topic_id: topic_record.id,
            space_member_id: space_member.id
          )

          if topic_member
            # 権限の判定
            can_update = topic_member.role_admin?
            can_create_page = true # メンバーであればページ作成可能
          end
        end
      end

      to_model(
        topic_record:,
        can_update:,
        can_create_page:
      )
    end
  end
end
