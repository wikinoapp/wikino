# typed: strict
# frozen_string_literal: true

module Views
end

Rails.autoloaders.main.push_dir(Rails.root.join("app/views").to_s, namespace: Views)
