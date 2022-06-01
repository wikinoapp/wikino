# typed: strict
# frozen_string_literal: true

class Note < ApplicationRecord
  extend T::Sig

  belongs_to :user

  has_one :content, class_name: "NoteContent", dependent: :destroy

  has_many :backlinks, class_name: "Link", dependent: :destroy, foreign_key: :target_note_id
  has_many :links, class_name: "Link", dependent: :destroy, foreign_key: :note_id
  has_many :referenced_notes, class_name: "Note", source: :note, through: :backlinks
  has_many :referencing_notes, class_name: "Note", through: :links, source: :target_note

  delegate :body, :body_html, to: :content, allow_nil: true

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

  sig { returns(T::Array[String]) }
  def titles_in_body
    body&.scan(%r{\[\[(.*?)\]\]})&.flatten || []
  end
end
