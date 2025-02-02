# typed: strict
# frozen_string_literal: true

class SpaceMember < ApplicationRecord
  include ModelConcerns::SpaceViewable

  enum :role, {
    SpaceMemberRole::Owner.serialize => 0
  }, prefix: true

  belongs_to :space
  belongs_to :user
  has_many :topic_memberships, dependent: :restrict_with_exception, foreign_key: :member_id, inverse_of: :member
  has_many :topics, through: :topic_memberships
  has_many :draft_pages, dependent: :restrict_with_exception, foreign_key: :editor_id, inverse_of: :editor
  has_many :page_editorships, dependent: :restrict_with_exception, foreign_key: :editor_id, inverse_of: :editor

  scope :active, -> { where(active: true) }
  scope :inactive, -> { where(active: false) }

  sig { params(topic: Topic, title: String).returns(Page) }
  def create_linked_page!(topic:, title:)
    page = space.not_nil!.pages.where(topic:, title:).first_or_create!(
      space:,
      body: "",
      body_html: "",
      linked_page_ids: [],
      modified_at: Time.zone.now
    )
    page_editorships.where(page:).first_or_create!(space:, last_page_modified_at: page.modified_at)

    page
  end

  sig { params(page: Page).returns(DraftPage) }
  def find_or_create_draft_page!(page:)
    draft_pages.create_with(
      space: page.space,
      topic: page.topic,
      title: page.title,
      body: page.body,
      body_html: page.body_html,
      linked_page_ids: page.linked_page_ids,
      modified_at: Time.zone.now
    ).find_or_create_by!(page:)
  rescue ActiveRecord::RecordNotUnique
    retry
  end

  sig { params(page: Page).void }
  def destroy_draft_page!(page:)
    draft_pages.where(page:).destroy_all

    nil
  end

  sig { returns(Page::PrivateAssociationRelation) }
  def last_modified_pages
    space.not_nil!.pages.joins(:editorships).merge(
      page_editorships.order(PageEditorship.arel_table[:last_page_modified_at].desc)
    )
  end

  sig { override.returns(Page::PrivateAssociationRelation) }
  def viewable_pages
    space.not_nil!.pages.active
  end

  sig { override.params(number: T.untyped).returns(Topic) }
  def find_topic_by_number!(number:)
    topics.find_by!(number:)
  end

  sig { override.params(topic: T.nilable(Topic)).returns(T::Boolean) }
  def can_create_page?(topic:)
    topic.present? && topics.where(id: topic.id).exists?
  end

  sig { override.returns(T::Boolean) }
  def can_create_topic?
    true
  end

  sig { override.params(page: Page).returns(T::Boolean) }
  def can_update_draft_page?(page:)
    topics.where(id: page.topic_id).exists?
  end
end
