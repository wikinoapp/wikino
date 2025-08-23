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
    return [] if page_records.empty?

    # 全ページのトピックとスペースを一括で取得
    topic_records = TopicRecord.where(id: page_records.map(&:topic_id).uniq).index_by(&:id)
    space_records = SpaceRecord.where(id: page_records.map(&:space_id).uniq).index_by(&:id)

    # current_space_memberのモデルを事前に作成
    current_space_member_model = if current_space_member
      SpaceMemberRepository.new.to_model(space_member_record: current_space_member)
    end

    # トピックごとにページをグループ化して処理
    pages_by_topic = page_records.group_by(&:topic_id)
    all_pages = []

    pages_by_topic.each do |topic_id, topic_pages|
      topic_record = topic_records[topic_id]
      next unless topic_record

      # このトピックのページが属するスペースを取得（通常は1つのスペース）
      first_page = topic_pages.first
      next unless first_page

      space_id = first_page.space_id
      space_record = space_records[space_id]
      next unless space_record

      topic = TopicRepository.new.to_model(topic_record:)
      space = SpaceRepository.new.to_model(space_record:)

      # Markupインスタンスを作成
      markup = Markup.new(
        current_topic: topic,
        current_space: space,
        current_space_member: current_space_member_model
      )

      # このトピックのページのbodyテキストを収集
      page_texts = topic_pages.map(&:body)

      # 一括でHTMLを生成
      body_htmls = markup.render_html_batch(texts: page_texts)

      # 各ページのモデルを生成
      topic_pages.each_with_index do |page_record, index|
        page_space = SpaceRepository.new.to_model(space_record: space_records[page_record.space_id].not_nil!)
        body_html = body_htmls[index]
        next unless body_html

        all_pages << Page.new(
          database_id: page_record.id,
          number: page_record.number,
          title: page_record.title,
          body: page_record.body,
          body_html:,
          modified_at: page_record.modified_at,
          published_at: page_record.published_at,
          pinned_at: page_record.pinned_at,
          trashed_at: page_record.trashed_at,
          can_update: nil,
          space: page_space,
          topic:
        )
      end
    end

    # 元の順序を保つため、IDでソート
    page_id_to_index = page_records.each_with_index.to_h { |pr, i| [pr.id, i] }
    all_pages.sort_by { |page| page_id_to_index[page.database_id] || Float::INFINITY }
  end
end
