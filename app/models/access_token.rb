# frozen_string_literal: true
# == Schema Information
#
# Table name: access_tokens
#
#  id         :uuid             not null, primary key
#  token      :string           not null
#  created_at :datetime         not null
#  updated_at :datetime         not null
#  user_id    :uuid             not null
#
# Indexes
#
#  index_access_tokens_on_token    (token) UNIQUE
#  index_access_tokens_on_user_id  (user_id) UNIQUE
#
# Foreign Keys
#
#  fk_rails_...  (user_id => users.id)
#
class AccessToken < ApplicationRecord
  has_secure_token :token

  belongs_to :user
end
