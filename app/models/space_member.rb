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

  sig { params(page: Page).void }
  def destroy_draft_page!(page:)
    draft_pages.where(page:).destroy_all

    nil
  end

  sig { override.returns(Page::PrivateRelation) }
  def viewable_pages
    space.pages.active
  end

  sig { override.returns(T::Boolean) }
  def can_create_topic?
    true
  end
end
