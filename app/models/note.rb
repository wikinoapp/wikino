# typed: false
# frozen_string_literal: true

class Note < ApplicationRecord
  belongs_to :user

  has_one :content, class_name: "NoteContent", dependent: :destroy

  has_many :backlinks, class_name: "Link", dependent: :destroy, foreign_key: :target_note_id
  has_many :links, class_name: "Link", dependent: :destroy, foreign_key: :note_id
  has_many :referenced_notes, class_name: "Note", source: :note, through: :backlinks
  has_many :referencing_notes, class_name: "Note", through: :links, source: :target_note

  validates :title, uniqueness: { scope: :user_id }

  def set_title!
    trimmed_title = content.body.split("\n").first&.strip&.delete_prefix("# ")
    self.title = trimmed_title.presence || "No Title @#{Time.zone.now.to_i}"
  end

  def link!
    titles = content.body.scan(%r{\[\[.*?\]\]}).map { |str| str.delete_prefix("[[").delete_suffix("]]") }

    target_note_ids = titles.map do |title|
      trimmed_title = title.strip
      user.notes.where(title: trimmed_title).first_or_create!.id
    end

    delete_note_ids = (referencing_notes.pluck(:id) - target_note_ids).uniq
    if delete_note_ids.present?
      links.where(target_note_id: delete_note_ids).destroy_all
    end

    (target_note_ids - referencing_notes.pluck(:id)).uniq.each do |target_note_id|
      links.where(note: self, target_note_id: target_note_id).first_or_create!
    end
  end
end
