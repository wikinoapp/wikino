# frozen_string_literal: true
# == Schema Information
#
# Table name: tags
#
#  id         :uuid             not null, primary key
#  name       :citext           not null
#  created_at :datetime         not null
#  updated_at :datetime         not null
#  user_id    :uuid             not null
#
# Indexes
#
#  index_tags_on_updated_at        (updated_at)
#  index_tags_on_user_id           (user_id)
#  index_tags_on_user_id_and_name  (user_id,name) UNIQUE
#
# Foreign Keys
#
#  fk_rails_...  (user_id => users.id)
#
class Tag < ApplicationRecord
  belongs_to :user

  has_many :taggings, dependent: :destroy
  has_many :notes, through: :taggings
end
