# typed: strict
# frozen_string_literal: true

class PageRepository < ApplicationRepository
  sig do
    params(
      page_record: PageRecord,
      can_update: T.nilable(T::Boolean),
      current_space_member: T.nilable(SpaceMemberRecord)
    ).returns(Page)
  end
  def to_model(page_record:, can_update: nil, current_space_member: nil)
    # body_html を動的に生成
    topic = TopicRepository.new.to_model(topic_record: page_record.topic_record.not_nil!)
    space = SpaceRepository.new.to_model(space_record: page_record.space_record.not_nil!)

    current_space_member_model = if current_space_member
      SpaceMemberRepository.new.to_model(space_member_record: current_space_member)
    end

    body_html = Markup.new(
      current_topic: topic,
      current_space: space,
      current_space_member: current_space_member_model
    ).render_html(text: page_record.body)

    Page.new(
      database_id: page_record.id,
      number: page_record.number,
      title: page_record.title,
      body: page_record.body,
      body_html:,
      modified_at: page_record.modified_at,
      published_at: page_record.published_at,
      pinned_at: page_record.pinned_at,
      trashed_at: page_record.trashed_at,
      can_update:,
      space:,
      topic:
    )
  end

  sig do
    params(
      page_records: T.any(
        T::Array[PageRecord],
        PageRecord::PrivateCollectionProxy,
        PageRecord::PrivateAssociationRelation,
        PageRecord::PrivateRelation
      ),
      current_space_member: T.nilable(SpaceMemberRecord)
    ).returns(T::Array[Page])
  end
  def to_models(page_records:, current_space_member: nil)
    page_records.map { to_model(page_record: it, current_space_member:) }
  end
end
