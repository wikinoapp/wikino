# typed: strict
# frozen_string_literal: true

class Page < ApplicationRecord
  include ModelConcerns::PageEditable

  acts_as_sequenced column: :number, scope: :space_id

  belongs_to :topic
  belongs_to :space
  has_many :editorships, class_name: "PageEditorship", dependent: :restrict_with_exception
  has_many :revisions, class_name: "PageRevision", dependent: :restrict_with_exception

  scope :published, -> { where.not(published_at: nil).where(archived_at: nil) }
  scope :initial, -> { where(title: nil) }

  # validates :body, length: {maximum: 1_000_000}
  # validates :original, absence: true

  # sig { returns(T.nilable(Page)) }
  # def original
  #   user&.pages_except(self)&.find_by(title:)
  # end

  sig { params(topic: Topic).returns(Page) }
  def self.create_as_initial!(topic:)
    initial.where(topic:).first_or_create!(
      space: topic.space,
      title: nil,
      body: "",
      body_html: "",
      linked_page_ids: [],
      modified_at: Time.zone.now
    )
  end

  sig { params(editor: User).void }
  def add_editor!(editor:)
    editorships.where(space:, editor:).first_or_create!(
      last_page_modified_at: modified_at
    )

    nil
  end

  sig { params(editor: User, body: String, body_html: String).returns(PageRevision) }
  def create_revision!(editor:, body:, body_html:)
    revisions.create!(space:, editor:, body:, body_html:)
  end
end
