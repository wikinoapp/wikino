# typed: strict
# frozen_string_literal: true

# Manual RBI file for Form classes that Sorbet cannot resolve

class ApplicationForm
  include ActiveModel::Validations
  include ActiveModel::Model
end

module TwoFactorAuths
  class CreationForm < ::ApplicationForm; end
  class DestructionForm < ::ApplicationForm; end  
  class RecoveryCodeRegenerationForm < ::ApplicationForm; end
end

module Profiles
  class EditForm < ::ApplicationForm; end
end

module Emails
  class EditForm < ::ApplicationForm; end
end