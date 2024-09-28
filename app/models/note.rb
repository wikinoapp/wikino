# typed: strict
# frozen_string_literal: true

class Note < ApplicationRecord
  include ModelConcerns::NoteEditable

  acts_as_sequenced column: :number, scope: :space_id

  belongs_to :author, class_name: "User"
  belongs_to :notebook
  belongs_to :space
  has_many :editorships, class_name: "NoteEditorship", dependent: :restrict_with_exception
  has_many :revisions, class_name: "NoteRevision", dependent: :restrict_with_exception

  scope :published, -> { where.not(published_at: nil).where(archived_at: nil) }
  scope :initial, -> { where(title: nil) }

  # validates :body, length: {maximum: 1_000_000}
  # validates :original, absence: true

  # sig { returns(T.nilable(Note)) }
  # def original
  #   user&.notes_except(self)&.find_by(title:)
  # end

  T::Sig::WithoutRuntime.sig { returns(Note::PrivateRelation) }
  def linked_notes
    Note.where(id: linked_note_ids)
  end

  T::Sig::WithoutRuntime.sig { returns(Note::PrivateRelation) }
  def backlinked_notes
    Note.where("'#{id}' = ANY (linked_note_ids)")
  end

  sig { returns(T::Array[String]) }
  def titles_in_body
    body.scan(%r{\[\[(.*?)\]\]}).flatten
  end

  sig { params(editor: User).void }
  def link!(editor:)
    linked_notes = titles_in_body.map do |title|
      editor.create_linked_note!(notebook: notebook.not_nil!, title:)
    end

    update!(linked_note_ids: linked_notes.pluck(:id))
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
