# frozen_string_literal: true

# == Schema Information
#
# Table name: users
#
#  id         :uuid             not null, primary key
#  deleted_at :datetime
#  email      :citext           not null
#  name       :string           default(""), not null
#  username   :citext           not null
#  created_at :datetime         not null
#  updated_at :datetime         not null
#
# Indexes
#
#  index_users_on_deleted_at  (deleted_at)
#  index_users_on_email       (email) UNIQUE
#  index_users_on_username    (username) UNIQUE
#
class User < ApplicationRecord
end
