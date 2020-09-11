# frozen_string_literal: true
# == Schema Information
#
# Table name: notes
#
#  id         :uuid             not null, primary key
#  body       :text             default(""), not null
#  body_html  :text             default(""), not null
#  title      :citext           default(""), not null
#  created_at :datetime         not null
#  updated_at :datetime         not null
#  user_id    :uuid             not null
#
# Indexes
#
#  index_notes_on_created_at         (created_at)
#  index_notes_on_updated_at         (updated_at)
#  index_notes_on_user_id            (user_id)
#  index_notes_on_user_id_and_title  (user_id,title) UNIQUE
#
# Foreign Keys
#
#  fk_rails_...  (user_id => users.id)
#
class Note < ApplicationRecord
  belongs_to :user

  has_many :backlinks, class_name: "Link", dependent: :destroy, foreign_key: :target_note_id
  has_many :links, class_name: "Link", dependent: :destroy, foreign_key: :note_id
  has_many :referenced_notes, class_name: "Note", source: :note, through: :backlinks
  has_many :referencing_notes, class_name: "Note", through: :links, source: :target_note

  validates :title, uniqueness: { scope: :user_id }

  def set_title!
    self.title = body.split("\n").first&.strip || "No Title - #{Time.zone.now.strftime('%Y-%m-%d %H:%M:%S.%L UTC')}"
  end

  def link!
    doc = Nokogiri::HTML(body_html)

    target_note_ids = doc.css("a[href*='#{ENV.fetch('NONOTO_URL')}']").each_with_object([]) do |element, ary|
      href = element.attr(:href)
      url_regex = %r{\A#{Regexp.escape(ENV.fetch('NONOTO_URL'))}/notes/(#{UUID_FORMAT})}

      next unless url_regex.match?(href)

      note_id = href.match(url_regex).captures.first
      target_note = user.notes.find_by(id: note_id)

      next unless target_note

      ary << target_note.id
    end

    delete_note_ids = (referencing_notes.pluck(:id) - target_note_ids).uniq
    if delete_note_ids.present?
      links.where(target_note_id: delete_note_ids).destroy_all
    end

    (target_note_ids - referencing_notes.pluck(:id)).uniq.each do |target_note_id|
      links.first_or_create!(note: self, target_note_id: target_note_id)
    end
  end
end
