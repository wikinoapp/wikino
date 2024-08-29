# typed: strict
# frozen_string_literal: true

class Note < ApplicationRecord
  acts_as_sequenced column: :number, scope: :space_id

  belongs_to :author, class_name: "User"
  belongs_to :list
  belongs_to :space
  has_many :backlinks, class_name: "NoteLink", dependent: :restrict_with_exception, foreign_key: :target_note_id
  has_many :links, class_name: "NoteLink", dependent: :restrict_with_exception, foreign_key: :source_note_id
  has_many :editorships, class_name: "NoteEditorship", dependent: :restrict_with_exception
  has_many :referenced_notes, class_name: "Note", source: :source_note, through: :backlinks
  has_many :referencing_notes, class_name: "Note", through: :links, source: :target_note
  has_many :revisions, class_name: "NoteRevision", dependent: :restrict_with_exception

  scope :published, -> { where.not(published_at: nil).where(archived_at: nil) }
  scope :initial, -> { where(title: nil) }

  # validates :body, length: {maximum: 1_000_000}
  # validates :original, absence: true

  # sig { returns(T.nilable(Note)) }
  # def original
  #   user&.notes_except(self)&.find_by(title:)
  # end

  sig { returns(T::Array[String]) }
  def titles_in_body
    body&.scan(%r{\[\[(.*?)\]\]})&.flatten || []
  end

  sig { params(editor: User).void }
  def link!(editor:)
    target_notes = titles_in_body.map do |title|
      editor.create_linked_note!(list:, title:)
    end
    target_note_ids = target_notes.pluck(:id)

    delete_note_ids = (referencing_notes.pluck(:id) - target_note_ids).uniq
    if delete_note_ids.present?
      links.where(target_note_id: delete_note_ids).destroy_all
    end

    (target_note_ids - referencing_notes.pluck(:id)).uniq.each do |target_note_id|
      links.where(target_note_id:).first_or_create!(space:)
    end
  end

  sig { params(editor: User).void }
  def add_editor!(editor:)
    editorships.where(space:, editor:).first_or_create!(
      last_note_modified_at: modified_at
    )
    
    nil
  end

  sig { params(editor: User, body: String, body_html: String).returns(NoteRevision) }
  def create_revision!(editor:, body:, body_html:)
    revisions.create!(space:, editor:, body:, body_html:)
  end
end
