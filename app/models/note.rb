# typed: strict
# frozen_string_literal: true

class Note < ApplicationRecord
  acts_as_sequenced column: :number, scope: :space_id

  belongs_to :author, class_name: "User"
  belongs_to :list
  belongs_to :space
  has_one :title, class_name: "NoteTitle", dependent: :restrict_with_exception
  has_many :backlinks, class_name: "Link", dependent: :restrict_with_exception, foreign_key: :target_note_id
  has_many :links, class_name: "Link", dependent: :restrict_with_exception
  has_many :editors, class_name: "NoteEditor", dependent: :restrict_with_exception
  has_many :referenced_notes, class_name: "Note", source: :note, through: :backlinks
  has_many :referencing_notes, class_name: "Note", through: :links, source: :target_note

  # validates :body, length: {maximum: 1_000_000}
  # validates :original, absence: true

  # sig { returns(T.nilable(Note)) }
  # def original
  #   user&.notes_except(self)&.find_by(title:)
  # end

  # sig { returns(String) }
  # def body_html
  #   render_html(body)
  # end

  sig { returns(T::Array[String]) }
  def titles_in_body
    body&.scan(%r{\[\[(.*?)\]\]})&.flatten || []
  end

  sig { void }
  def link!
    target_note_ids = titles_in_body.map do |title|
      T.must(user).notes.where(title:).first_or_create!.id
    end

    delete_note_ids = (referencing_notes.pluck(:id) - target_note_ids).uniq
    if delete_note_ids.present?
      links.where(target_note_id: delete_note_ids).destroy_all
    end

    (target_note_ids - referencing_notes.pluck(:id)).uniq.each do |target_note_id|
      links.where(note: self, target_note_id: target_note_id).first_or_create!
    end
  end

  sig { params(user: User).returns(NoteEditor) }
  def add_editor(user:)
    editors.where(space:, user:).first_or_create!(
      last_note_modified_at: modified_at
    )
  end

  sig { params(text: String).returns(String) }
  private def render_html(text)
    GitHub::Markup.render_s(
      GitHub::Markups::MARKUP_MARKDOWN,
      text,
      options: {commonmarker_opts: %i[HARDBREAKS]}
    )
  end
end
