# frozen_string_literal: true

# == Schema Information
#
# Table name: oauth_providers
#
#  id               :uuid             not null, primary key
#  name             :string           not null
#  token            :string           not null
#  token_expires_at :integer
#  uid              :string           not null
#  created_at       :datetime         not null
#  updated_at       :datetime         not null
#  user_id          :uuid             not null
#
# Indexes
#
#  index_oauth_providers_on_name_and_uid  (name,uid) UNIQUE
#  index_oauth_providers_on_user_id       (user_id)
#
# Foreign Keys
#
#  fk_rails_...  (user_id => users.id)
#
class OauthProvider < ApplicationRecord
  extend Enumerize

  enumerize :name, in: %i(google)

  belongs_to :user
end
