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
    topic = TopicRepository.new.to_model(topic_record: page_record.topic_record.not_nil!)
    space = SpaceRepository.new.to_model(space_record: page_record.space_record.not_nil!)

    Page.new(
      database_id: page_record.id,
      number: page_record.number,
      title: page_record.title,
      body: page_record.body,
      body_html: page_record.body_html,
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
    return [] if page_records.empty?

    # 全ページのトピックとスペースを一括で取得
    topic_records = TopicRecord.where(id: page_records.map(&:topic_id).uniq).index_by(&:id)
    space_records = SpaceRecord.where(id: page_records.map(&:space_id).uniq).index_by(&:id)

    all_pages = []

    page_records.each do |page_record|
      topic_record = topic_records[page_record.topic_id]
      next unless topic_record

      space_record = space_records[page_record.space_id]
      next unless space_record

      topic = TopicRepository.new.to_model(topic_record:)
      space = SpaceRepository.new.to_model(space_record:)

      all_pages << Page.new(
        database_id: page_record.id,
        number: page_record.number,
        title: page_record.title,
        body: page_record.body,
        body_html: page_record.body_html,
        modified_at: page_record.modified_at,
        published_at: page_record.published_at,
        pinned_at: page_record.pinned_at,
        trashed_at: page_record.trashed_at,
        can_update: nil,
        space:,
        topic:
      )
    end

    all_pages
  end
end
