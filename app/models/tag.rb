# frozen_string_literal: true
# == Schema Information
#
# Table name: tags
#
#  id         :bigint           not null, primary key
#  name       :citext           not null
#  created_at :datetime         not null
#  updated_at :datetime         not null
#  project_id :bigint           not null
#
# Indexes
#
#  index_tags_on_project_id           (project_id)
#  index_tags_on_project_id_and_name  (project_id,name) UNIQUE
#  index_tags_on_updated_at           (updated_at)
#
# Foreign Keys
#
#  fk_rails_...  (project_id => projects.id)
#
class Tag < ApplicationRecord
  belongs_to :project

  has_many :taggings, dependent: :destroy
  has_many :notes, through: :taggings
end
