# typed: strict
# frozen_string_literal: true

class PageRecord < ApplicationRecord
  include Discard::Model

  include RecordConcerns::Pageable

  self.table_name = "pages"

  acts_as_sequenced column: :number, scope: :space_id

  belongs_to :topic_record, foreign_key: :topic_id
  belongs_to :space_record, foreign_key: :space_id
  has_many :draft_page_records,
    dependent: :restrict_with_exception,
    foreign_key: :page_id,
    inverse_of: :page_record
  has_many :page_editor_records,
    class_name: "PageEditorRecord",
    dependent: :restrict_with_exception,
    foreign_key: :page_id,
    inverse_of: :page_record
  has_many :revision_records,
    class_name: "PageRevisionRecord",
    dependent: :restrict_with_exception,
    foreign_key: :page_id,
    inverse_of: :page_record
  has_many :page_attachment_reference_records,
    class_name: "PageAttachmentReferenceRecord",
    dependent: :restrict_with_exception,
    foreign_key: :page_id,
    inverse_of: :page_record

  scope :published, -> { where.not(published_at: nil) }
  scope :pinned, -> { where.not(pinned_at: nil) }
  scope :not_pinned, -> { where(pinned_at: nil) }
  scope :not_trashed, -> { where(trashed_at: nil) }
  scope :topics_kept, -> { joins(:topic_record).merge(TopicRecord.kept) }
  scope :topics_visibility_public, -> { joins(:topic_record).merge(TopicRecord.visibility_public) }
  scope :visible, -> { kept.topics_kept }
  scope :available, -> { visible.not_trashed }
  scope :active, -> { available.published }
  scope :restorable, -> { where(trashed_at: Page::DELETE_LIMIT_DAYS.days.ago..) }
  scope :filter_by_title, ->(q:) {
    words = q&.split.presence || []
    conditions = words.map.with_index { |_, i| "(title ILIKE :word#{i})" }.join(" AND ")
    parameters = words.map.with_index { |word, i| {"word#{i}": "%#{word}%"} }.reduce({}, :merge)
    where(conditions, parameters)
  }

  sig { params(topic_record: TopicRecord).returns(PageRecord) }
  def self.create_as_blanked!(topic_record:)
    # 空のbody_htmlを生成
    topic = TopicRepository.new.to_model(topic_record:)
    space = SpaceRepository.new.to_model(space_record: topic_record.space_record.not_nil!)

    body_html = Markup.new(
      current_topic: topic,
      current_space: space,
      current_space_member: nil
    ).render_html(text: "")

    topic_record.page_records.create!(
      space_record: topic_record.space_record,
      title: nil,
      body: "",
      body_html:,
      linked_page_ids: [],
      modified_at: Time.current
    )
  end

  sig { void }
  def self.destroy_all_with_related_records!
    find_each do |page_record|
      page_record.draft_page_records.delete_all(:delete_all)
      page_record.page_editor_records.delete_all(:delete_all)
      page_record.revision_records.delete_all(:delete_all)
      page_record.page_attachment_reference_records.delete_all(:delete_all)
    end

    delete_all

    nil
  end

  sig { returns(SpaceRecord) }
  def space_record!
    space_record.not_nil!
  end

  sig { returns(TopicRecord) }
  def topic_record!
    topic_record.not_nil!
  end

  sig { returns(T::Boolean) }
  def pinned?
    pinned_at.present?
  end

  sig { returns(T::Boolean) }
  def published?
    published_at.present?
  end

  sig { returns(T::Boolean) }
  def trashed?
    trashed_at.present?
  end

  sig do
    params(
      user_record: T.nilable(UserRecord)
    ).returns(
      T.any(
        PageRecord::PrivateAssociationRelationWhereChain,
        PageRecord::PrivateAssociationRelation
      )
    )
  end
  def backlinked_page_records(user_record:)
    pages = space_record.not_nil!.page_records.available.where("'#{id}' = ANY (linked_page_ids)")
    topic_records = user_record.nil? ? TopicRecord.visibility_public : user_record.viewable_topics

    pages.joins(:topic_record).merge(topic_records)
  end

  # 全スペース内のページを検索（参加スペース + 公開トピック）
  sig { params(user_record: UserRecord, keyword: String).returns(PageRecord::PrivateRelation) }
  def self.search_in_user_spaces(user_record:, keyword:)
    keywords = keyword.split.compact_blank
    return none if keywords.empty?

    # 参加しているスペースのページのID
    member_query = joins(:space_record)
      .joins("INNER JOIN space_members ON spaces.id = space_members.space_id")
      .where("space_members.user_id = ? AND space_members.active = ?", user_record.id, true)
      .active
      .order(modified_at: :desc)
      .limit(25)

    # 各キーワードにAND検索を適用
    keywords.each do |kw|
      member_query = member_query.where("pages.title ILIKE ?", "%#{kw}%")
    end
    member_page_ids = member_query.pluck(:id)

    # 公開トピックのページのID（参加していないスペースも含む）
    public_query = joins(:topic_record)
      .where(topics: {visibility: TopicVisibility::Public.serialize})
      .active
      .order(modified_at: :desc)
      .limit(25)

    # 各キーワードにAND検索を適用
    keywords.each do |kw|
      public_query = public_query.where("pages.title ILIKE ?", "%#{kw}%")
    end
    public_page_ids = public_query.pluck(:id)

    # IDを結合して重複を除去
    combined_ids = (member_page_ids + public_page_ids).uniq

    # 結果を取得
    where(id: combined_ids).active.order(modified_at: :desc)
  end

  # 指定されたスペース内のページを検索
  sig do
    params(
      user_record: UserRecord,
      space_identifiers: T::Array[String],
      keywords: T::Array[String]
    ).returns(PageRecord::PrivateRelation)
  end
  def self.search_in_specific_spaces(user_record:, space_identifiers:, keywords:)
    # 参加しているスペースのページのクエリ
    member_query = joins(:space_record)
      .joins("INNER JOIN space_members ON spaces.id = space_members.space_id")
      .where("space_members.user_id = ? AND space_members.active = ?", user_record.id, true)
      .where(spaces: {identifier: space_identifiers})
      .active
      .order(modified_at: :desc)
      .limit(25)

    # 各キーワードにAND検索を適用
    keywords.each do |keyword|
      member_query = member_query.where("pages.title ILIKE ?", "%#{keyword}%")
    end
    member_page_ids = member_query.pluck(:id)

    # 指定されたスペースの公開トピックのページのクエリ
    public_query = joins(:space_record, :topic_record)
      .where(spaces: {identifier: space_identifiers})
      .where(topics: {visibility: TopicVisibility::Public.serialize})
      .active
      .order(modified_at: :desc)
      .limit(25)

    # 各キーワードにAND検索を適用
    keywords.each do |keyword|
      public_query = public_query.where("pages.title ILIKE ?", "%#{keyword}%")
    end
    public_page_ids = public_query.pluck(:id)

    # IDを結合して重複を除去
    combined_ids = (member_page_ids + public_page_ids).uniq

    # 結果を取得
    where(id: combined_ids).active.order(modified_at: :desc)
  end

  sig { params(editor_record: SpaceMemberRecord).void }
  def add_editor!(editor_record:)
    page_editor_records.where(space_record:, space_member_record: editor_record).first_or_create!(
      last_page_modified_at: modified_at
    )

    nil
  end

  sig do
    params(
      editor_record: SpaceMemberRecord,
      body: String,
      body_html: String
    ).returns(PageRevisionRecord)
  end
  def create_revision!(editor_record:, body:, body_html:)
    revision_records.create!(space_record:, space_member_record: editor_record, body:, body_html:)
  end

  # ページ本文から添付ファイルIDを抽出し、page_attachment_referencesレコードを更新
  sig { params(body: String).void }
  def update_attachment_references!(body:)
    # 本文から添付ファイルのIDを抽出
    attachment_ids = extract_attachment_ids(body)

    # 現在の参照を取得
    current_attachment_ids = page_attachment_reference_records.pluck(:attachment_id)

    # 新しく追加される添付ファイル
    new_attachment_ids = attachment_ids - current_attachment_ids

    # 削除される添付ファイル
    removed_attachment_ids = current_attachment_ids - attachment_ids

    # 新しい参照を作成
    new_attachment_ids.each do |attachment_id|
      # 添付ファイルが実際に存在するか確認
      if AttachmentRecord.exists?(id: attachment_id, space_id:)
        PageAttachmentReferenceRecord.create!(
          page_id: id,
          attachment_id:
        )
      end
    end

    # 不要な参照を削除
    if removed_attachment_ids.any?
      PageAttachmentReferenceRecord.where(
        page_id: id,
        attachment_id: removed_attachment_ids
      ).destroy_all
    end

    nil
  end

  # 本文から添付ファイルIDを抽出
  sig { params(body: String).returns(T::Array[String]) }
  private def extract_attachment_ids(body)
    attachment_ids = T.let([], T::Array[String])

    # imgタグのsrc属性から抽出
    # <img src="/attachments/attachment_id">
    img_pattern = %r{<img[^>]+src=["'](/attachments/([^/"']+))["'][^>]*>}
    body.scan(img_pattern) do |_full_url, attachment_id|
      attachment_ids << attachment_id if attachment_id
    end

    # aタグのhref属性から抽出
    # <a href="/attachments/attachment_id">
    link_pattern = %r{<a[^>]+href=["'](/attachments/([^/"']+))["'][^>]*>}
    body.scan(link_pattern) do |_full_url, attachment_id|
      attachment_ids << attachment_id if attachment_id
    end

    # Markdown形式の画像から抽出
    # ![alt text](/attachments/attachment_id)
    markdown_img_pattern = %r{!\[[^\]]*\]\(/attachments/([^/)]+)\)}
    body.scan(markdown_img_pattern) do |match|
      # scanの戻り値は配列なので、最初の要素を取得
      attachment_ids << match[0] if match[0]
    end

    # Markdown形式のリンクから抽出
    # [link text](/attachments/attachment_id)
    markdown_link_pattern = %r{(?<!!)\[[^\]]+\]\(/attachments/([^/)]+)\)}
    body.scan(markdown_link_pattern) do |match|
      # scanの戻り値は配列なので、最初の要素を取得
      attachment_ids << match[0] if match[0]
    end

    # 重複を削除
    attachment_ids.uniq
  end

  belongs_to :featured_image_attachment_record,
    class_name: "AttachmentRecord",
    foreign_key: :featured_image_attachment_id,
    optional: true

  # Markdown本文の1行目から画像IDを抽出（featured画像として使用）
  sig { returns(T.nilable(String)) }
  def extract_featured_image_id
    return nil if body.blank?

    # 1行目を取得
    first_line = body.lines.first&.strip
    return nil if first_line.blank?

    # 1. Markdown画像形式をチェック: ![alt text](/attachments/attachment_id)
    markdown_match = first_line.match(%r{!\[[^\]]*\]\(/attachments/([^)]+)\)})
    return markdown_match[1] if markdown_match

    # 2. HTML img要素をチェック: <img src="/attachments/attachment_id" ...>
    img_match = first_line.match(%r{<img[^>]+src=[\"']/attachments/([^\"']+)[\"'][^>]*>}i)
    return img_match[1] if img_match

    nil
  end

  # featured画像がGIFかどうか判定
  sig { returns(T::Boolean) }
  def featured_image_is_gif?
    attachment = featured_image_attachment_record
    return false unless attachment

    filename = attachment.filename
    return false unless filename

    filename.downcase.end_with?(".gif")
  end

  # カード用画像URLを取得
  sig { params(expires_in: ActiveSupport::Duration).returns(T.nilable(String)) }
  def card_image_url(expires_in: 1.hour)
    attachment = featured_image_attachment_record
    return nil unless attachment

    # GIFの場合はオリジナル画像のURLを返す
    if featured_image_is_gif?
      attachment.generate_signed_url(space_member_record: nil, expires_in:)
    else
      attachment.thumbnail_url(size: AttachmentThumbnailSize::Card, expires_in:)
    end
  end

  # OGP画像URLを取得
  sig { params(expires_in: ActiveSupport::Duration).returns(T.nilable(String)) }
  def og_image_url(expires_in: 1.hour)
    attachment = featured_image_attachment_record
    return nil unless attachment

    # GIFの場合はnilを返す（デフォルトOGP画像を使用）
    return nil if featured_image_is_gif?

    attachment.thumbnail_url(size: AttachmentThumbnailSize::Og, expires_in:)
  end
end
